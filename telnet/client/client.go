package client

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	cmd "telnet/command"
	"telnet/connection"
	opt "telnet/option"
	"telnet/terminal"
)

type Client struct {
	connection.Connection
	InputLength int
}

func (c *Client) Call() {
	// Request TELNET Commands
	err := c.ReqCmds(c.SupportOptions)
	if err != nil {
		c.ErrChan <- err
	}

	// Open tty
	c.Terminal = terminal.New()
	err = c.Terminal.OpenTty()
	if err != nil {
		c.ErrChan <- err
	}
	defer c.Terminal.Close()

	// Catch signal
	go c.CatchSignal()
	// Scan key input and write message
	go c.ScanAndWrite()

	// Read server message
	for {
		byteMessage, err := c.ReadMessage()
		if err == io.EOF {
			fmt.Println("Connection closed by foreign host.")
			os.Exit(0)
		} else if err != nil {
			c.ErrChan <- err
		}
		if byteMessage != nil {
			fmt.Print(string(byteMessage))
		}
	}
}

func (c *Client) ScanAndWrite() {
	ttyReader := bufio.NewReader(c.Terminal.StdFile)
	c.InputLength = 0
	for {
		r, _, err := ttyReader.ReadRune()
		if err != nil {
			c.ErrChan <- err
			return
		}
		if !c.EnableOptions[opt.ECHO] {
			switch r {
			case '\r', '\n':
				c.InputLength = 0
				fmt.Print("\n")
			case '\177':
				if c.InputLength > 0 {
					fmt.Print("\b \b")
					c.InputLength--
				}
			default:
				fmt.Print(string(r))
				c.InputLength++
			}
		}
		err = c.WriteByte(byte(r))
		if err != nil {
			c.ErrChan <- err
			return
		}
	}
}

func (c *Client) CatchSignal() {
	var err error
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGWINCH)
	for {
		switch <-signalChan {
		case os.Interrupt:
			err = c.WriteByte(3)
		case syscall.SIGWINCH:
			byteNAWSReq, err := BuildCmdRes(c.Connection, cmd.DO, opt.NEGOTIATE_ABOUT_WINDOW_SIZE)
			if err != nil {
				c.ErrChan <- err
			}
			err = c.WriteBytes(byteNAWSReq)
		}
		if err != nil {
			c.ErrChan <- err
		}
	}
}

func New(ip string, port int, supportOptions []byte) *Client {
	c := new(Client)
	c.IP = ip
	c.Port = port
	c.EnableOptions = map[byte]bool{}
	c.SupportOptions = supportOptions
	c.BuildCmdRes = BuildCmdRes
	c.InputLength = 0
	return c
}

func Run(ip string, port int) {
	// New TELNET Client
	supportOptions := []byte{opt.ECHO, opt.NEGOTIATE_ABOUT_WINDOW_SIZE, opt.TERMINAL_SPEED, opt.TERMINAL_TYPE, opt.SUPPRESS_GO_AHEAD}
	c := New(ip, port, supportOptions)
	// Handle errors
	c.ErrChan = make(chan error)
	defer close(c.ErrChan)
	go func() {
		for err := range c.ErrChan {
			log.Fatal("Error:", err)
		}
	}()

	// TCP Dial
	fmt.Printf("Trying %s:%d...\n", ip, port)
	err := c.Dial()
	if err != nil {
		c.ErrChan <- err
	}
	defer c.Conn.Close()
	fmt.Printf("Connected to %s:%d.\n", ip, port)

	// TELNET Call
	c.Call()
}

func BuildCmdRes(c connection.Connection, mainCmd byte, subCmd byte, options ...byte) ([]byte, error) {
	var err error
	bufCmdsRes := new(bytes.Buffer)
	nextStatus := false

	if !cmd.IsNeedOption(mainCmd) {
		switch mainCmd {
		case cmd.SB:
			if !c.IsSupportOption(subCmd) {
				_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
				nextStatus = false
				break
			}
			SEND := byte(1)
			IS := byte(0)
			if options[0] != SEND {
				break
			}
			bufOptionRes := bytes.NewBuffer([]byte{cmd.IAC, cmd.SB, subCmd, IS})
			switch subCmd {
			case opt.TERMINAL_SPEED:
				bufOptionRes.Write([]byte(strconv.Itoa(int(c.Terminal.Termios.Ospeed)) + "," + strconv.Itoa(int(c.Terminal.Termios.Ispeed))))
				_, err = bufCmdsRes.Write(bufOptionRes.Bytes())
			case opt.TERMINAL_TYPE:
				bufOptionRes.Write([]byte(strings.ToUpper(c.Terminal.Type)))
				_, err = bufCmdsRes.Write(bufOptionRes.Bytes())
			}
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.SE})
		}

		_, ok := c.EnableOptions[subCmd]
		if !ok {
			c.EnableOptions[subCmd] = nextStatus
		}
		return bufCmdsRes.Bytes(), err
	}

	switch mainCmd {
	case cmd.WILL:
		if !c.IsSupportOption(subCmd) {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
			nextStatus = false
			break
		}
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DO, subCmd})
		nextStatus = true
	case cmd.WONT:
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
		nextStatus = false
	case cmd.DO:
		if !c.IsSupportOption(subCmd) || subCmd == opt.ECHO {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
			nextStatus = c.IsSupportOption(subCmd) && subCmd == opt.ECHO
			break
		}
		if !c.EnableOptions[subCmd] {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WILL, subCmd})
			nextStatus = true
		}
		switch subCmd {
		case opt.NEGOTIATE_ABOUT_WINDOW_SIZE:
			height, width, _ := c.Terminal.GetSize()
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.SB, opt.NEGOTIATE_ABOUT_WINDOW_SIZE})
			binary.Write(bufCmdsRes, binary.BigEndian, int16(width))
			binary.Write(bufCmdsRes, binary.BigEndian, int16(height))
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.SE})
		}
	case cmd.DONT:
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
		nextStatus = false
	}

	status, ok := c.EnableOptions[subCmd]
	if ok && status == nextStatus && subCmd != opt.ECHO {
		return nil, err
	}
	c.EnableOptions[subCmd] = nextStatus
	return bufCmdsRes.Bytes(), err
}

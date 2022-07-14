package client

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	cmd "telnet/command"
	"telnet/connection"

	"github.com/mattn/go-tty"
	"golang.org/x/crypto/ssh/terminal"
)

type Client struct {
	connection.Connection
	InputLength int
}

func (c *Client) Call() error {
	conn, err := net.Dial("tcp", c.IP+":"+strconv.Itoa(c.Port))
	c.Conn = conn
	c.Reader = bufio.NewReader(conn)
	return err
}

func (c *Client) BuildCmdsRes(mainCmd byte, subCmd byte, options ...byte) ([]byte, error) {
	bufCmdsRes := new(bytes.Buffer)
	var err error
	switch mainCmd {
	case cmd.SB:
		if !c.IsSupportOption(subCmd) {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
			c.EnableOptions[subCmd] = false
			break
		}
		SEND := byte(1)
		IS := byte(0)
		if options[0] != SEND {
			break
		}
		bufOptionRes := bytes.NewBuffer([]byte{cmd.IAC, cmd.SB, subCmd, IS})
		switch subCmd {
		case OPTION_TERMINAL_SPEED:
			bufOptionRes.Write([]byte("38400,38400"))
			_, err = bufCmdsRes.Write(bufOptionRes.Bytes())
		case OPTION_TERMINAL_TYPE:
			bufOptionRes.Write([]byte("XTERM-256COLOR"))
			_, err = bufCmdsRes.Write(bufOptionRes.Bytes())
		}
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.SE})
	case cmd.WILL:
		if !c.IsSupportOption(subCmd) {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
			c.EnableOptions[subCmd] = false
			break
		}
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DO, subCmd})
		c.EnableOptions[subCmd] = true
	case cmd.WONT:
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
		c.EnableOptions[subCmd] = false
	case cmd.DO:
		if !c.IsSupportOption(subCmd) {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
			c.EnableOptions[subCmd] = false
			break
		}
		if !c.EnableOptions[subCmd] {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WILL, subCmd})
			c.EnableOptions[subCmd] = true
		}
		switch subCmd {
		case OPTION_NEGOTIATE_ABOUT_WINDOW_SIZE:
			width, hight, _ := terminal.GetSize(syscall.Stdin)
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.SB, OPTION_NEGOTIATE_ABOUT_WINDOW_SIZE})
			binary.Write(bufCmdsRes, binary.BigEndian, int16(width))
			binary.Write(bufCmdsRes, binary.BigEndian, int16(hight))
		}
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.SE})
	case cmd.DONT:
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
		c.EnableOptions[subCmd] = false
	}
	return bufCmdsRes.Bytes(), err
}

func (c *Client) Read() ([]byte, error) {
	var err error
	var byteResCmd []byte
	byteMessage, err := c.ReadAll()
	if err != nil {
		return nil, err
	}
	subCmd := byte(0)
	i := -1
	optionStartIndex := -1
	bufMessage := new(bytes.Buffer)
	bufCmdsRes := new(bytes.Buffer)
	for i < len(byteMessage)-1 {
		i++
		b := byteMessage[i]
		if b == cmd.IAC {
			i++
			mainCmd := byteMessage[i]
			// subnegotiation
			switch mainCmd {
			case cmd.SB:
				subCmd = byteMessage[i+1]
				optionStartIndex = i + 2
				i += 2
				continue
			case cmd.SE:
				byteResCmd, err = c.BuildCmdsRes(cmd.SB, subCmd, byteMessage[optionStartIndex:i-1]...)
				_, err = bufCmdsRes.Write(byteResCmd)
				if err != nil {
					return nil, err
				}
				optionStartIndex = -1
			}
			// commands
			if !cmd.IsNeedOption(mainCmd) {
				byteResCmd, err = c.BuildCmdsRes(mainCmd, 0)
				_, err = bufCmdsRes.Write(byteResCmd)
			} else {
				i++
				subCmd = byteMessage[i]
				byteResCmd, err = c.BuildCmdsRes(mainCmd, subCmd)
				_, err = bufCmdsRes.Write(byteResCmd)
			}
			if err != nil {
				return nil, err
			}
			continue
		}
		// subnegotiation
		if optionStartIndex >= 0 {
			continue
		}
		err = bufMessage.WriteByte(b)
		if err != nil {
			return nil, err
		}
	}
	c.WriteBytes(bufCmdsRes.Bytes())
	return bufMessage.Bytes(), err
}

func (c *Client) ScanAndWrite(tty *tty.TTY) error {
	c.InputLength = 0
	for {
		r, err := tty.ReadRune()
		if err != nil {
			return err
		}
		if !c.EnableOptions[OPTION_ECHO] {
			switch r {
			case '\r':
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
			return err
		}
	}
}

func (c *Client) CatchSignal() {
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	for {
		<-quit
		err := c.WriteByte(3)
		if err != nil {
			log.Fatal("Write Error:", err)
		}
	}
}

func Init(ip string, port int, supportOptions []byte) Client {
	c := Client{}
	c.IP = ip
	c.Port = port
	c.EnableOptions = map[byte]bool{}
	c.InputLength = 0
	c.SupportOptions = supportOptions
	return c
}

func Run(ip string, port int) {
	tty, err := tty.Open()
	if err != nil {
		log.Fatal("Open Error:", err)
	}
	defer tty.Close()

	// Init TELNET Client
	supportOptions := []byte{OPTION_ECHO, OPTION_NEGOTIATE_ABOUT_WINDOW_SIZE, OPTION_TERMINAL_SPEED, OPTION_TERMINAL_TYPE}
	c := Init(ip, port, supportOptions)

	// TCP Call
	fmt.Printf("Trying %s:%d...\n", ip, port)
	err = c.Call()
	if err != nil {
		log.Fatal("Call Error:", err)
	}
	defer c.Conn.Close()
	fmt.Printf("Connected to %s:%d.\n", ip, port)

	// Request TELNET Commands
	err = c.ReqCmds(supportOptions)
	if err != nil {
		log.Fatal("Write Error:", err)
	}

	// Catch interrupt signal
	go c.CatchSignal()
	// Scan key input and write message
	go c.ScanAndWrite(tty)

	// Read server message
	for {
		byteMessage, err := c.Read()
		if err == io.EOF {
			fmt.Println("Connection closed by foreign host.")
			os.Exit(0)
		} else if err != nil {
			log.Fatal("Read Error:", err)
		}
		if byteMessage != nil {
			fmt.Print(string(byteMessage))
		}
	}
}

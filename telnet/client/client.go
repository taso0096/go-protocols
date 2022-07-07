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
	CmdFlag       bool
	Cmd           byte
	SubCmd        byte
	EnableOptions map[byte]bool
	InputLength   int
}

func (c *Client) Call() error {
	conn, err := net.Dial("tcp", c.IP+":"+strconv.Itoa(c.Port))
	c.Conn = conn
	c.Reader = bufio.NewReader(conn)
	return err
}

func (c *Client) InitCmd() {
	c.CmdFlag = false
	c.Cmd = 0
	c.SubCmd = 0
}

func (c *Client) ExecCmd() error {
	var err error
	switch c.Cmd {
	case cmd.WILL:
		if IsSupportOption(c.SubCmd) {
			err = c.Write([]byte{cmd.IAC, cmd.DO, c.SubCmd})
			c.EnableOptions[c.SubCmd] = true
		} else {
			err = c.Write([]byte{cmd.IAC, cmd.DONT, c.SubCmd})
			c.EnableOptions[c.SubCmd] = false
		}
	case cmd.WONT:
		err = c.Write([]byte{cmd.IAC, cmd.WONT, c.SubCmd})
		c.EnableOptions[c.SubCmd] = false
	case cmd.DO:
		if IsSupportOption(c.SubCmd) {
			err = c.Write([]byte{cmd.IAC, cmd.WILL, c.SubCmd})
			c.EnableOptions[c.SubCmd] = true
			switch c.SubCmd {
			case OPTION_NEGOTIATE_ABOUT_WINDOW_SIZE:
				width, hight, _ := terminal.GetSize(syscall.Stdin)
				optionDetail := []byte{cmd.IAC, cmd.SB, OPTION_NEGOTIATE_ABOUT_WINDOW_SIZE}
				bufWindowSize := new(bytes.Buffer)
				binary.Write(bufWindowSize, binary.BigEndian, int16(width))
				binary.Write(bufWindowSize, binary.BigEndian, int16(hight))
				err = c.Write(append(optionDetail, bufWindowSize.Bytes()...))
			}
		} else {
			err = c.Write([]byte{cmd.IAC, cmd.WONT, c.SubCmd})
			c.EnableOptions[c.SubCmd] = false
		}
	case cmd.DONT:
		err = c.Write([]byte{cmd.IAC, cmd.DONT, c.SubCmd})
		c.EnableOptions[c.SubCmd] = false
	}
	return err
}

func (c *Client) Read() ([]byte, error) {
	byteMessage, err := c.ReadAll()
	if err != nil {
		return nil, err
	}
	startIndex := 0
	for i, b := range append(byteMessage, 0) {
		if b == cmd.IAC {
			c.CmdFlag = true
			continue
		} else if c.CmdFlag {
			if c.Cmd == 0 {
				c.Cmd = b
				if !cmd.IsNeedOption(b) {
					c.ExecCmd()
					c.InitCmd()
				}
				continue
			}
			c.SubCmd = b
			c.ExecCmd()
			c.InitCmd()
			continue
		} else if i == len(byteMessage) {
			return nil, nil
		}
		startIndex = i
		break
	}
	return byteMessage[startIndex:], err
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

func Init(ip string, port int) Client {
	c := Client{}
	c.IP = ip
	c.Port = port
	c.EnableOptions = map[byte]bool{}
	c.InputLength = 0
	c.InitCmd()
	return c
}

func Run(ip string, port int) {
	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	c := Init(ip, port)

	fmt.Printf("Trying %s:%d...\n", ip, port)
	err = c.Call()
	if err != nil {
		log.Fatal("Call Error:", err)
	}
	defer c.Conn.Close()
	fmt.Printf("Connected to %s:%d.\n", ip, port)

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	go func() {
		for {
			<-quit
			err = c.WriteByte(3)
			if err != nil {
				log.Fatal("Write Error:", err)
			}
		}
	}()

	for {
		byteMessage, err := c.Read()
		if err == io.EOF {
			fmt.Println("Connection closed by foreign host.")
			os.Exit(0)
		} else if err != nil {
			log.Fatal("Read Error:", err)
		}
		if byteMessage == nil {
			continue
		}
		fmt.Print(string(byteMessage))
		go c.ScanAndWrite(tty)
	}
}

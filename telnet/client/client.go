package client

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
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
	for {
		r, err := tty.ReadRune()
		if err != nil {
			return err
		}
		if r == '\n' {
			return c.Write([]byte("\r\n"))
		}
		err = c.Write([]byte{byte(r)})
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

	for {
		byteMessage, err := c.Read()
		if err != nil {
			log.Fatal("Read Error:", err)
		}
		if byteMessage == nil {
			continue
		}
		fmt.Print(string(byteMessage))
		go c.ScanAndWrite(tty)
	}
}

package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	cmd "telnet/command"
	"telnet/connection"

	"github.com/mattn/go-tty"
)

type Client struct {
	connection.Connection
	CmdFlag bool
	Cmd     byte
	SubCmd  byte
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
	case cmd.DO:
		switch c.SubCmd {
		case cmd.OPTION_ECHO:
			err = c.Write([]byte{cmd.IAC, cmd.WILL, c.SubCmd})
		default:
			err = c.Write([]byte{cmd.IAC, cmd.WONT, c.SubCmd})
		}
	case cmd.WILL:
		switch c.SubCmd {
		case cmd.OPTION_ECHO:
			err = c.Write([]byte{cmd.IAC, cmd.DO, c.SubCmd})
		default:
			err = c.Write([]byte{cmd.IAC, cmd.DONT, c.SubCmd})
		}
	}
	return err
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
		b, err := c.ReadByte()
		if err != nil {
			log.Fatal("Read Error:", err)
		}
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
		}
		byteMessage, err := c.ReadAll()
		if err != nil {
			log.Fatal("Read Error:", err)
		}
		message := string(append([]byte{b}, byteMessage...))
		fmt.Print(message)
		go c.ScanAndWrite(tty)
	}
}

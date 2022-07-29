package connection

import (
	"bufio"
	"bytes"
	"net"
	"os/exec"
	"strconv"
	cmd "telnet/command"
	opt "telnet/option"

	"telnet/terminal"
)

type Connection struct {
	// TCP Config
	IP     string
	Port   int
	Conn   net.Conn
	Reader *bufio.Reader
	// TELNET Config
	IsServer       bool
	SupportOptions []byte
	EnableOptions  map[byte]bool
	// Build TELNET Command Response Function
	BuildCmdRes func(c Connection, mainCmd byte, subCmd byte, options ...byte) ([]byte, error)
	// exec.Cmd for pty
	ExecCmdChan chan *exec.Cmd
	// Channel for error handle
	ErrChan chan error
	// Terminal Config
	Terminal *terminal.Terminal
}

func (c *Connection) Accept(ln net.Listener) error {
	conn, err := ln.Accept()
	if err != nil {
		return err
	}
	c.Conn = conn
	c.Reader = bufio.NewReader(conn)
	return nil
}

func (c *Connection) Dial() error {
	conn, err := net.Dial("tcp", c.IP+":"+strconv.Itoa(c.Port))
	if err != nil {
		return err
	}
	c.Conn = conn
	c.Reader = bufio.NewReader(conn)
	return nil
}

func (c *Connection) WriteByte(message byte) error {
	_, err := c.Conn.Write([]byte{message})
	return err
}

func (c *Connection) WriteBytes(message []byte) error {
	_, err := c.Conn.Write(message)
	return err
}

func (c *Connection) ReadMessage() ([]byte, error) {
	var err error
	var byteCmdRes []byte
	byteMessage, err := c.ReadAll()
	if err != nil {
		return nil, err
	}
	subCmd := byte(0)
	optionStartIndex := -1
	bufMessage := new(bytes.Buffer)
	bufCmdsRes := new(bytes.Buffer)
	for i := 0; i < len(byteMessage); i++ {
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
				byteCmdRes, err = c.BuildCmdRes(*c, cmd.SB, subCmd, byteMessage[optionStartIndex:i-1]...)
				_, err = bufCmdsRes.Write(byteCmdRes)
				if err != nil {
					return nil, err
				}
				optionStartIndex = -1
			}
			// commands
			if !cmd.IsNeedOption(mainCmd) {
				byteCmdRes, err = c.BuildCmdRes(*c, mainCmd, 0)
				_, err = bufCmdsRes.Write(byteCmdRes)
			} else {
				i++
				subCmd = byteMessage[i]
				byteCmdRes, err = c.BuildCmdRes(*c, mainCmd, subCmd)
				_, err = bufCmdsRes.Write(byteCmdRes)
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

func (c *Connection) ReadByte() (byte, error) {
	return c.Reader.ReadByte()
}

func (c *Connection) ReadBytes(length int) ([]byte, error) {
	message := make([]byte, length)
	n, err := c.Reader.Read(message)
	return message[:n], err
}

func (c *Connection) ReadAll() ([]byte, error) {
	message := make([]byte, c.Reader.Size())
	n, err := c.Reader.Read(message)
	return message[:n], err
}

func (c *Connection) IsSupportOption(option byte) bool {
	for _, v := range c.SupportOptions {
		if option == v {
			return true
		}
	}
	return false
}

func (c *Connection) ReqCmds(subCmds []byte) error {
	bufReqCmds := new(bytes.Buffer)
	for _, subCmd := range subCmds {
		if c.IsServer {
			if subCmd == opt.ECHO || subCmd == opt.TERMINAL_TYPE || subCmd == opt.NEGOTIATE_ABOUT_WINDOW_SIZE || subCmd == opt.TERMINAL_SPEED {
				bufReqCmds.Write([]byte{cmd.IAC, cmd.DO, subCmd})
				continue
			}
		} else {
			if subCmd == opt.ECHO {
				bufReqCmds.Write([]byte{cmd.IAC, cmd.DO, subCmd})
				continue
			}
		}
		bufReqCmds.Write([]byte{cmd.IAC, cmd.WILL, subCmd})
	}
	err := c.WriteBytes(bufReqCmds.Bytes())
	if err != nil {
		return err
	}
	for _, subCmd := range subCmds {
		c.EnableOptions[subCmd] = true
	}
	return nil
}

package connection

import (
	"bufio"
	"bytes"
	"net"
	"os"
	"os/exec"
	cmd "telnet/command"
	opt "telnet/option"
)

type Connection struct {
	// TCP Config
	IP     string
	Port   int
	Conn   net.Conn
	Reader *bufio.Reader
	// TELNET Config
	SupportOptions []byte
	EnableOptions  map[byte]bool
	// Build TELNET Command Response Function
	BuildCmdRes func(c Connection, mainCmd byte, subCmd byte, options ...byte) ([]byte, error)
	// pty in TELNET Server
	Ptmx        *os.File
	ExecCmdChan chan *exec.Cmd
	// Channel for error handle
	ErrChan chan error
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
		if subCmd == opt.ECHO || subCmd == opt.TERMINAL_TYPE || subCmd == opt.NEGOTIATE_ABOUT_WINDOW_SIZE {
			bufReqCmds.Write([]byte{cmd.IAC, cmd.DO, subCmd})
			continue
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

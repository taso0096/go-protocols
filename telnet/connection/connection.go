package connection

import (
	"bufio"
	"bytes"
	"net"
	cmd "telnet/command"
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
}

func (c *Connection) WriteByte(message byte) error {
	_, err := c.Conn.Write([]byte{message})
	return err
}

func (c *Connection) WriteBytes(message []byte) error {
	_, err := c.Conn.Write(message)
	return err
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
	cmdsBuffer := bytes.NewBuffer([]byte{})
	for _, subCmd := range subCmds {
		cmdsBuffer.Write([]byte{cmd.IAC, cmd.WILL, subCmd})
	}
	err := c.WriteBytes(cmdsBuffer.Bytes())
	if err != nil {
		return err
	}
	for _, subCmd := range subCmds {
		c.EnableOptions[subCmd] = true
	}
	return nil
}

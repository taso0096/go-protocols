package connection

import (
	"bufio"
	"net"
)

type Connection struct {
	// TCP Config
	IP     string
	Port   int
	Conn   net.Conn
	Reader *bufio.Reader
	// TELNET Config
	SupportOptions []byte
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

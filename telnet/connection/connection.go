package connection

import (
	"bufio"
	"net"
)

type Connection struct {
	IP     string
	Port   int
	Conn   net.Conn
	Reader *bufio.Reader
}

func (c *Connection) Write(message []byte) error {
	_, err := c.Conn.Write(message)
	return err
}

func (c *Connection) ReadByte() (byte, error) {
	return c.Reader.ReadByte()
}

func (c *Connection) ReadAll() ([]byte, error) {
	message := make([]byte, c.Reader.Size())
	n, err := c.Reader.Read(message)

	return message[:n], err
}

func (c *Connection) Read(length int) ([]byte, error) {
	message := make([]byte, length)
	n, err := c.Reader.Read(message)

	return message[:n], err
}

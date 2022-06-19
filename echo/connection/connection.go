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

func (c *Connection) Write(message string) error {
	_, err := c.Conn.Write([]byte(message))
	return err
}

func (c *Connection) Read() (string, error) {
	return c.Reader.ReadString('\x00')
}

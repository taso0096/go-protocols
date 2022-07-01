package connection

import (
	"bufio"
	"bytes"
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
	var p [1]byte
	var buffer bytes.Buffer

	for {
		n, err := c.Reader.Read(p[:])
		if n > 0 {
			buffer.Write(p[:n])
		}
		if p[0] == byte(0) || err != nil {
			break
		}
	}

	return buffer.ReadString('\x00')
}

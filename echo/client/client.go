package client

import (
	"bufio"
	"net"
	"strconv"
)

type Client struct {
	IP     string
	Port   int
	Conn   net.Conn
	Reader *bufio.Reader
}

func (c *Client) Call() error {
	conn, err := net.Dial("tcp", c.IP+":"+strconv.Itoa(c.Port))
	c.Conn = conn
	c.Reader = bufio.NewReader(conn)
	return err
}

func (c *Client) Write(message string) error {
	_, err := c.Conn.Write([]byte(message))
	return err
}

func (c *Client) Read() (string, error) {
	return c.Reader.ReadString('\x00')
}

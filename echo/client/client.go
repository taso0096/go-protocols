package client

import (
	"bufio"
	"connection"
	"net"
	"strconv"
)

type Client struct {
	connection.Connection
}

func (c *Client) Call() error {
	conn, err := net.Dial("tcp", c.IP+":"+strconv.Itoa(c.Port))
	c.Conn = conn
	c.Reader = bufio.NewReader(conn)
	return err
}

func Init(ip string, port int) Client {
	client := Client{}
	client.IP = ip
	client.Port = port
	return client
}

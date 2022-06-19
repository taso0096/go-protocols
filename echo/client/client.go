package client

import (
	"bufio"
	"connection"
	"fmt"
	"log"
	"net"
	"os"
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

func Run(ip string, port int) {
	client := Init(ip, port)

	err := client.Call()
	if err != nil {
		log.Fatal("Call Error:", err)
	}
	defer client.Conn.Close()

	fmt.Print("> ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	err = client.Write(scanner.Text() + "\x00")
	if err != nil {
		log.Fatal("Write Error:", err)
	}

	message, err := client.Read()
	if err != nil {
		log.Fatal("Read Error:", err)
	}
	log.Println(message)
}

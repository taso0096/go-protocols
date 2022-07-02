package client

import (
	"log"
	"telnet/connection"
)

type Client struct {
	connection.Connection
}

func Run(ip string, port int) {
	log.Println(ip, port)
}

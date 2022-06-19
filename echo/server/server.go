package server

import (
	"bufio"
	"connection"
	"net"
	"strconv"
)

type Server struct {
	connection.Connection
}

func (h *Server) Listen() error {
	ln, err := net.Listen("tcp", h.IP+":"+strconv.Itoa(h.Port))
	if err != nil {
		return err
	}
	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		return err
	}

	h.Conn = conn
	h.Reader = bufio.NewReader(conn)
	return nil
}

func Init(ip string, port int) Server {
	client := Server{}
	client.IP = ip
	client.Port = port
	return client
}

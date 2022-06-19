package server

import (
	"bufio"
	"net"
	"strconv"
)

type Server struct {
	IP     string
	Port   int
	Conn   net.Conn
	Reader *bufio.Reader
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

func (h *Server) Write(message string) error {
	_, err := h.Conn.Write([]byte(message))
	return err
}

func (h *Server) Read() (string, error) {
	return h.Reader.ReadString('\x00')
}

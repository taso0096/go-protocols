package server

import (
	"bufio"
	"connection"
	"io"
	"log"
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
	server := Server{}
	server.IP = ip
	server.Port = port
	return server
}

func Run(ip string, port int) {
	server := Init(ip, port)

	for {
		err := server.Listen()
		if err != nil {
			log.Fatal("Listen Error:", err)
		}
		defer server.Conn.Close()

		message, err := server.Read()
		if err != nil && err != io.EOF {
			log.Fatal("Read Error:", err)
		}
		log.Println(message)

		err = server.Write(message)
		if err != nil {
			log.Fatal("Write Error:", err)
		}
	}
}

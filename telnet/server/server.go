package server

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	cmd "telnet/command"
	"telnet/connection"
)

type Server struct {
	connection.Connection
}

func (s *Server) Listen() error {
	ln, err := net.Listen("tcp", s.IP+":"+strconv.Itoa(s.Port))
	if err != nil {
		return err
	}
	defer ln.Close()

	conn, err := ln.Accept()
	if err != nil {
		return err
	}

	s.Conn = conn
	s.Reader = bufio.NewReader(conn)
	return nil
}

func Init(ip string, port int, supportOptions []byte) Server {
	s := *new(Server)
	s.IP = ip
	s.Port = port
	s.EnableOptions = map[byte]bool{}
	s.SupportOptions = supportOptions
	s.BuildCmdRes = BuildCmdRes
	return s
}

func Run(ip string, port int) {
	// Init TELNET Server
	supportOptions := []byte{OPTION_ECHO}
	s := Init(ip, port, supportOptions)

	fmt.Printf("Listen start in %s:%d...\n", ip, port)
	err := s.Listen()
	if err != nil {
		log.Fatal("Listen Error:", err)
	}
	defer s.Conn.Close()

	// Request TELNET Commands
	err = s.ReqCmds(supportOptions)
	if err != nil {
		log.Fatal("Write Error:", err)
	}

	for {
		byteMessage, err := s.ReadMessage()
		if err == io.EOF {
			fmt.Println("Connection closed by foreign host.")
			os.Exit(0)
		} else if err != nil {
			log.Fatal("Read Error:", err)
		}
		if byteMessage != nil {
			fmt.Print(string(byteMessage))
		}
	}
}

func BuildCmdRes(c connection.Connection, mainCmd byte, subCmd byte, options ...byte) ([]byte, error) {
	bufCmdsRes := new(bytes.Buffer)
	var err error
	switch mainCmd {
	case cmd.WILL:
		if !c.IsSupportOption(subCmd) {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
			c.EnableOptions[subCmd] = false
			break
		}
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DO, subCmd})
		c.EnableOptions[subCmd] = true
	case cmd.WONT:
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
		c.EnableOptions[subCmd] = false
	case cmd.DO:
		if !c.IsSupportOption(subCmd) {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
			c.EnableOptions[subCmd] = false
			break
		}
		if !c.EnableOptions[subCmd] {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WILL, subCmd})
			c.EnableOptions[subCmd] = true
		}
	case cmd.DONT:
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
		c.EnableOptions[subCmd] = false
	}
	return bufCmdsRes.Bytes(), err
}

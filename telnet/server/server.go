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
	BufParsedMessage bytes.Buffer
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

func (s *Server) ListenAndHandle() {
	s.Reset()
	err := s.Listen()
	if err != nil {
		log.Fatal("Listen Error:", err)
	}
	defer s.Conn.Close()
	fmt.Printf("Connected.\n")

	// Request TELNET Commands
	err = s.ReqCmds(s.SupportOptions)
	if err != nil {
		log.Fatal("Write Error:", err)
	}

	s.WriteBytes([]byte("> "))
	for {
		byteMessage, err := s.ReadMessage()
		if err == io.EOF {
			fmt.Println("Connection closed by foreign host.")
			os.Exit(0)
		} else if err != nil {
			log.Fatal("Read Error:", err)
		}
		if byteMessage == nil {
			continue
		}
		byteParsedMessage, err := s.ParseMessage(byteMessage)
		if byteParsedMessage == nil {
			continue
		}
		log.Println(string(byteParsedMessage))
		if string(byteParsedMessage) == "exit" {
			break
		}
		s.WriteBytes(append(byteParsedMessage, []byte{'\r', '\n'}...))
		s.WriteBytes([]byte("> "))
	}
	s.Conn.Close()
}

func (s *Server) ParseMessage(byteMessage []byte) ([]byte, error) {
	var byteParsedMessage []byte
	bufEchoMessage := new(bytes.Buffer)
	i := -1
SCAN_BYTE_MESSAGE:
	for i < len(byteMessage)-1 {
		i++
		b := byteMessage[i]
		switch b {
		case '\r', '\n':
			byteParsedMessage = s.BufParsedMessage.Bytes()
			s.BufParsedMessage.Reset()
			if s.EnableOptions[OPTION_ECHO] {
				s.WriteBytes([]byte{'\r', '\n'})
			}
			break SCAN_BYTE_MESSAGE
		default:
			s.BufParsedMessage.WriteByte(b)
			bufEchoMessage.WriteByte(b)
		}
	}
	if s.EnableOptions[OPTION_ECHO] {
		s.WriteBytes(bufEchoMessage.Bytes())
	}
	return byteParsedMessage, nil
}

func (s *Server) Reset() {
	s.EnableOptions = map[byte]bool{}
	s.BufParsedMessage = *new(bytes.Buffer)
}

func Init(ip string, port int, supportOptions []byte) Server {
	s := *new(Server)
	s.IP = ip
	s.Port = port
	s.SupportOptions = supportOptions
	s.BuildCmdRes = BuildCmdRes
	s.Reset()
	return s
}

func Run(ip string, port int) {
	// Init TELNET Server
	supportOptions := []byte{OPTION_ECHO}
	s := Init(ip, port, supportOptions)

	fmt.Printf("Listen on %s:%d...\n", ip, port)
	for {
		s.ListenAndHandle()
	}
}

func BuildCmdRes(c connection.Connection, mainCmd byte, subCmd byte, options ...byte) ([]byte, error) {
	var err error
	bufCmdsRes := new(bytes.Buffer)
	nextStatus := false
	switch mainCmd {
	case cmd.WILL:
		if !c.IsSupportOption(subCmd) {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
			nextStatus = false
			break
		}
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DO, subCmd})
		nextStatus = true
	case cmd.WONT:
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
		nextStatus = false
	case cmd.DO:
		if !c.IsSupportOption(subCmd) {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
			nextStatus = false
			break
		}
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WILL, subCmd})
		nextStatus = true
	case cmd.DONT:
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
		nextStatus = false
	}

	status, ok := c.EnableOptions[subCmd]
	if ok && status == nextStatus {
		return nil, nil
	}
	c.EnableOptions[subCmd] = nextStatus
	return bufCmdsRes.Bytes(), err
}

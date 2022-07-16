package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strconv"
	cmd "telnet/command"
	"telnet/connection"
	opt "telnet/option"

	"github.com/creack/pty"
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

func (s *Server) ListenAndHandle() error {
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

	// Start pty
	bash := exec.Command("bash", "-c", "stty -echo && bash")
	s.Ptmx, err = pty.Start(bash)
	if err != nil {
		log.Fatal("pty.Start Error:", err)
	}
	defer s.Ptmx.Close()
	// Writes pty results to TELNET connection
	go func() {
		byteResult := make([]byte, 4096)
		for {
			n, err := s.Ptmx.Read(byteResult)
			if err != nil {
				s.Conn.Close()
				return
			}
			s.WriteBytes(byteResult[:n])
		}
	}()

	for {
		byteMessage, err := s.ReadMessage()
		if err != nil {
			return err
		}
		if byteMessage == nil {
			continue
		}
		byteParsedMessage, err := s.ParseMessage(byteMessage)
		if byteParsedMessage == nil {
			continue
		}
		log.Print(string(byteParsedMessage))
		s.Ptmx.WriteString(string(byteParsedMessage))
	}
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
			s.BufParsedMessage.Write([]byte("\n"))
			byteParsedMessage = s.BufParsedMessage.Bytes()
			s.BufParsedMessage.Reset()
			if s.EnableOptions[opt.ECHO] {
				s.WriteBytes([]byte("\r\n"))
			}
			break SCAN_BYTE_MESSAGE
		case '\177':
			if s.BufParsedMessage.Len() == 0 {
				continue
			}
			s.BufParsedMessage.Truncate(s.BufParsedMessage.Len() - 1)
			if s.EnableOptions[opt.ECHO] {
				s.WriteBytes([]byte("\b \b"))
			}
		default:
			s.BufParsedMessage.WriteByte(b)
			bufEchoMessage.WriteByte(b)
		}
	}
	if s.EnableOptions[opt.ECHO] {
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
	supportOptions := []byte{opt.ECHO, opt.SUPPRESS_GO_AHEAD, opt.NEGOTIATE_ABOUT_WINDOW_SIZE}
	s := Init(ip, port, supportOptions)

	fmt.Printf("Listen on %s:%d...\n", ip, port)
	for {
		err := s.ListenAndHandle()
		log.Println(err)
	}
}

func BuildCmdRes(c connection.Connection, mainCmd byte, subCmd byte, options ...byte) ([]byte, error) {
	var err error
	bufCmdsRes := new(bytes.Buffer)
	nextStatus := false

	if !cmd.IsNeedOption(mainCmd) {
		switch mainCmd {
		case cmd.SB:
			if !c.IsSupportOption(subCmd) {
				_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
				nextStatus = false
				break
			}
			switch subCmd {
			case opt.NEGOTIATE_ABOUT_WINDOW_SIZE:
				if len(options) != 4 {
					break
				}
				err := pty.Setsize(c.Ptmx, &pty.Winsize{
					Rows: binary.BigEndian.Uint16(options[2:4]),
					Cols: binary.BigEndian.Uint16(options[0:2]),
				})
				if err != nil {
					return nil, err
				}
			}
		}

		_, ok := c.EnableOptions[subCmd]
		if !ok {
			c.EnableOptions[subCmd] = nextStatus
		}
		return bufCmdsRes.Bytes(), err
	}

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
		if subCmd == opt.ECHO {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WILL, opt.ECHO})
			break
		}
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
		nextStatus = false
	case cmd.DO:
		if !c.IsSupportOption(subCmd) {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
			nextStatus = false
			break
		} else if subCmd != opt.ECHO {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WILL, subCmd})
		}
		nextStatus = true
	case cmd.DONT:
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DONT, subCmd})
		nextStatus = false
	}

	status, ok := c.EnableOptions[subCmd]
	if ok && status == nextStatus {
		return nil, err
	}
	c.EnableOptions[subCmd] = nextStatus
	return bufCmdsRes.Bytes(), err
}

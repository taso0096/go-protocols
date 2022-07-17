package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"strconv"
	"strings"
	cmd "telnet/command"
	"telnet/connection"
	opt "telnet/option"

	"github.com/creack/pty"
)

type Server struct {
	connection.Connection
	BufEchoMessage bytes.Buffer
}

func (s *Server) Accept(ln net.Listener) error {
	conn, err := ln.Accept()
	if err != nil {
		return err
	}
	s.Conn = conn
	s.Reader = bufio.NewReader(conn)
	return nil
}

func (s *Server) Handle(ln net.Listener, errChan chan<- error) {
	err := s.Accept(ln)
	if err != nil {
		errChan <- err
		return
	}
	fmt.Printf("Client Connected.\n")

	go func() {
		defer s.Conn.Close()

		// Request TELNET Commands
		err = s.ReqCmds(s.SupportOptions)
		if err != nil {
			errChan <- err
			return
		}

		// Start pty
		bash := exec.Command("login")
		s.Ptmx, err = pty.Start(bash)
		if err != nil {
			errChan <- err
			return
		}
		defer s.Ptmx.Close()
		// Relay output from pty to client
		go s.ReadPty(errChan)

		// Relay input from client to pty
		for {
			byteMessage, err := s.ReadMessage()
			if err != nil {
				errChan <- err
				break
			}
			if byteMessage == nil {
				continue
			}
			if !s.EnableOptions[opt.ECHO] {
				s.BufEchoMessage.Write(byteMessage)
			}
			s.Ptmx.Write(byteMessage)
		}
	}()
}

func (s *Server) ReadPty(errChan chan<- error) {
	startIndex := 0
	byteResult := make([]byte, 4096)
	for {
		n, err := s.Ptmx.Read(byteResult)
		if err != nil {
			errChan <- err
			s.Conn.Close()
			return
		}
		// Exclude client input if echo option is not enabled
		if !s.EnableOptions[opt.ECHO] {
			startIndex = 0
			for i := 0; i < n; i++ {
				b, err := s.BufEchoMessage.ReadByte()
				if b == '\177' {
					startIndex = n
					break
				}
				if err == io.EOF {
					break
				} else if err != nil {
					errChan <- err
					s.Conn.Close()
					return
				}
				if strings.Contains("\r\n", string(b)) && strings.Contains("\r\n", string(byteResult[i])) {
					i++
				} else if b != byteResult[i] {
					break
				}
				startIndex = i + 1
			}
			s.BufEchoMessage.Reset()
		}
		if startIndex < n {
			s.WriteBytes(byteResult[startIndex:n])
		}
	}
}

func Init(ip string, port int, supportOptions []byte) Server {
	s := *new(Server)
	s.IP = ip
	s.Port = port
	s.SupportOptions = supportOptions
	s.BuildCmdRes = BuildCmdRes
	s.EnableOptions = map[byte]bool{}
	s.BufEchoMessage = *new(bytes.Buffer)
	return s
}

func Run(ip string, port int) {
	// Listen TCP
	ln, err := net.Listen("tcp", ip+":"+strconv.Itoa(port))
	if err != nil {
		log.Fatal("Listen Error:", err)
	}
	defer ln.Close()
	fmt.Printf("Listen on %s:%d...\n", ip, port)

	// Logging errors
	errChan := make(chan error, 2)
	defer close(errChan)
	go func() {
		for err := range errChan {
			log.Println("Handle Error:", err)
		}
	}()

	// Handle connections
	supportOptions := []byte{opt.ECHO, opt.SUPPRESS_GO_AHEAD, opt.NEGOTIATE_ABOUT_WINDOW_SIZE}
	for {
		s := Init(ip, port, supportOptions)
		s.Handle(ln, errChan)
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

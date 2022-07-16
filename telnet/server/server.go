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
	errChan := make(chan error, 2)
	s.Reset()
	err := s.Listen()
	if err != nil {
		return err
	}
	defer s.Conn.Close()
	fmt.Printf("Connected.\n")

	// Request TELNET Commands
	err = s.ReqCmds(s.SupportOptions)
	if err != nil {
		return err
	}

	// Start pty
	bash := exec.Command("login")
	s.Ptmx, err = pty.Start(bash)
	if err != nil {
		return err
	}
	defer s.Ptmx.Close()
	// Writes pty results to TELNET connection
	go func() {
		startIndex := 0
		byteResult := make([]byte, 4096)
		for {
			n, err := s.Ptmx.Read(byteResult)
			if err != nil {
				errChan <- err
				s.Conn.Close()
				return
			}
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
						startIndex++
						i++
					} else if b != byteResult[i] {
						break
					}
					startIndex++
				}
				s.BufEchoMessage.Reset()
			}
			if startIndex < n {
				s.WriteBytes(byteResult[startIndex:n])
			}
		}
	}()

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
	return <-errChan
}

func (s *Server) Reset() {
	s.EnableOptions = map[byte]bool{}
	s.BufEchoMessage = *new(bytes.Buffer)
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
		log.Println("ListenAndHandle Error:", err)
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

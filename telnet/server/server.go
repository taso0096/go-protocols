package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	cmd "telnet/command"
	"telnet/connection"
	opt "telnet/option"
	"telnet/terminal"
)

type Server struct {
	connection.Connection
	BufEchoMessage bytes.Buffer
}

func (s *Server) Handle(ln net.Listener) {
	err := s.Accept(ln)
	if err != nil {
		s.ErrChan <- err
		return
	}
	log.Println("Client Connected")

	// Start pty
	s.Terminal = terminal.New()
	go func() {
		s.Terminal.EnvChan = make(chan []string)
		defer close(s.Terminal.EnvChan)
		for env := range s.Terminal.EnvChan {
			err = s.Terminal.StartPty(env)
			if err != nil {
				s.ErrChan <- err
			}
			// Relay output from pty to client
			s.ReadPty()
		}
	}()

	// Read client message
	go func() {
		defer s.Conn.Close()
		defer s.Terminal.Close()

		// Request TELNET Commands
		err = s.ReqCmds(s.SupportOptions)
		if err != nil {
			s.ErrChan <- err
			return
		}

		// Relay input from client to pty
		for {
			byteMessage, err := s.ReadMessage()
			if err != nil {
				s.ErrChan <- err
				break
			}
			if byteMessage == nil {
				continue
			}
			if !s.EnableOptions[opt.ECHO] {
				s.BufEchoMessage.Write(byteMessage)
			}
			s.Terminal.Write(byteMessage)
		}
	}()
}

func (s *Server) ReadPty() {
	startIndex := 0
	byteResult := make([]byte, 4096)
	for {
		n, err := s.Terminal.Read(byteResult)
		if err != nil {
			s.ErrChan <- err
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
					s.ErrChan <- err
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

func New(ip string, port int, supportOptions []byte) *Server {
	s := new(Server)
	s.IsServer = true
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
		log.Fatal("Error:", err)
	}
	defer ln.Close()
	fmt.Printf("Listen on %s:%d...\n", ip, port)

	// Logging errors
	errChan := make(chan error, 2)
	defer close(errChan)
	go func() {
		for err := range errChan {
			log.Println("Error:", err)
		}
	}()

	// Handle connections
	supportOptions := []byte{opt.ECHO, opt.SUPPRESS_GO_AHEAD, opt.TERMINAL_TYPE, opt.NEGOTIATE_ABOUT_WINDOW_SIZE, opt.TERMINAL_SPEED}
	for {
		s := New(ip, port, supportOptions)
		s.ErrChan = errChan
		s.Handle(ln)
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
			case opt.TERMINAL_TYPE:
				IS := byte(0)
				if options[0] != IS {
					break
				}
				if c.Terminal.StdFile != nil {
					return nil, fmt.Errorf("pty already opened")
				}
				c.Terminal.SetType(strings.ToLower(string(options[1:])))
			case opt.NEGOTIATE_ABOUT_WINDOW_SIZE:
				if len(options) != 4 {
					break
				}
				err := c.Terminal.SetSize(binary.BigEndian.Uint16(options[0:2]), binary.BigEndian.Uint16(options[2:4]))
				if err != nil {
					return nil, err
				}
			case opt.TERMINAL_SPEED:
				IS := byte(0)
				if options[0] != IS {
					break
				}
				speeds := strings.Split(string(options[1:]), ",")
				ospeed, _ := strconv.Atoi(speeds[0])
				ispeed, _ := strconv.Atoi(speeds[1])
				c.Terminal.SetSpeed(ospeed, ispeed)
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
		if !c.EnableOptions[subCmd] {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.DO, subCmd})
			nextStatus = true
		}
		switch subCmd {
		case opt.TERMINAL_TYPE, opt.TERMINAL_SPEED:
			SEND := byte(1)
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.SB, subCmd, SEND})
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.SE})
		}
	case cmd.WONT:
		if subCmd == opt.ECHO {
			_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WILL, opt.ECHO})
			break
		}
		_, err = bufCmdsRes.Write([]byte{cmd.IAC, cmd.WONT, subCmd})
		nextStatus = false
		switch subCmd {
		case opt.TERMINAL_TYPE:
			if c.Terminal.StdFile != nil {
				return nil, fmt.Errorf("pty already opened")
			}
			c.Terminal.SetType("vt100")
		}
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

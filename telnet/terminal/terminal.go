package terminal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"reflect"

	"github.com/creack/pty"
	"github.com/mattn/go-tty"
	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

func New() *Terminal {
	return new(Terminal)
}

type Terminal struct {
	StdFile *os.File
	Termios unix.Termios
	Type    string
	EnvChan chan []string
	// go-tty
	tty *tty.TTY
	// Window Size
	width  uint16
	height uint16
	// Baud Rate
	ispeed int
	ospeed int
	// StdFile Reader
	reader *bufio.Reader
}

func (t *Terminal) OpenTty() error {
	tty, err := tty.Open()
	if err != nil {
		return err
	}
	t.Type = os.Getenv("TERM")
	if len(t.Type) == 0 {
		t.Type = "VT100"
	}
	t.tty = tty
	t.StdFile = tty.Input() // StdIn
	t.reader = bufio.NewReader(t.StdFile)
	termios.Tcgetattr(tty.Input().Fd(), &t.Termios)
	return nil
}

func (t *Terminal) StartPty(env []string) error {
	execCmd := exec.Command("login")
	execCmd.Env = env
	ptmx, err := pty.Start(execCmd)
	if err != nil {
		return err
	}
	t.StdFile = ptmx // StdIn + StdOut + StdErr
	t.reader = bufio.NewReader(t.StdFile)
	termios.Tcgetattr(t.StdFile.Fd(), &t.Termios)
	if t.width > 0 && t.height > 0 {
		t.setsize()
	}
	if t.ospeed > 0 && t.ispeed > 0 {
		t.setspeed()
	}
	return nil
}

func (t *Terminal) SetType(terminalType string) {
	t.Type = terminalType
	t.EnvChan <- append(os.Environ(), "TERM="+terminalType)
}

func (t *Terminal) GetSize() (height int, width int, err error) {
	return pty.Getsize(t.StdFile)
}

func (t *Terminal) SetSize(width uint16, height uint16) error {
	t.width = width
	t.height = height
	return t.setsize()
}

func (t *Terminal) setsize() error {
	if t.StdFile == nil {
		return nil
	}
	err := pty.Setsize(t.StdFile, &pty.Winsize{
		Rows: t.height,
		Cols: t.width,
	})
	return err
}

func (t *Terminal) SetSpeed(ospeed int, ispeed int) {
	t.ospeed = ospeed
	t.ispeed = ispeed
	t.setspeed()
}

func (t *Terminal) setspeed() {
	if t.StdFile == nil {
		return
	}
	ospeedRv := reflect.ValueOf(&t.Termios.Ospeed)
	ispeedRv := reflect.ValueOf(&t.Termios.Ispeed)
	ospeedRv.Elem().SetUint(uint64(t.ispeed))
	ispeedRv.Elem().SetUint(uint64(t.ispeed))
	termios.Tcsetattr(t.StdFile.Fd(), termios.TCSANOW, &t.Termios)
}

func (t *Terminal) Close() error {
	var err error
	if t.tty != nil {
		err = t.tty.Close()
	} else if t.StdFile != nil {
		err = t.StdFile.Close()
	}
	return err
}

func (t *Terminal) Read(p []byte) (n int, err error) {
	if t.reader == nil {
		return 0, fmt.Errorf("Not set bufio.Reader in Terminal")
	}
	return t.reader.Read(p)
}

func (t *Terminal) ReadRune() (r rune, size int, err error) {
	if t.reader == nil {
		return 0, 0, fmt.Errorf("Not set bufio.Reader in Terminal")
	}
	return t.reader.ReadRune()
}

func (t *Terminal) Write(b []byte) (n int, err error) {
	if t.StdFile == nil {
		return 0, fmt.Errorf("Not set StdFile in Terminal")
	}
	return t.StdFile.Write(b)
}

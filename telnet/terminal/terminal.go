package terminal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"syscall"

	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

type Terminal struct {
	StdFile *os.File
	Termios unix.Termios
	Type    string
	EnvChan chan []string
	// Window Size
	width  uint16
	height uint16
	// Baud Rate
	ispeed int
	ospeed int
	// StdFile Reader
	reader *bufio.Reader
}

func New() *Terminal {
	return new(Terminal)
}

func (t *Terminal) OpenTty() error {
	ttyStdin, err := os.Open("/dev/tty")
	if err != nil {
		return err
	}
	t.Type = os.Getenv("TERM")
	if len(t.Type) == 0 {
		t.Type = "VT100"
	}
	t.StdFile = ttyStdin // Stdin
	t.reader = bufio.NewReader(t.StdFile)
	termios.Tcgetattr(t.StdFile.Fd(), &t.Termios)
	t.setRawMode()
	return nil
}

func (t *Terminal) StartPty(env []string) error {
	// Set OS command for login
	cmd := exec.Command("login")
	cmd.Env = env
	// Open pty
	pty, tty, err := termios.Pty()
	if err != nil {
		return err
	}
	defer tty.Close()
	cmd.Stdin = tty
	cmd.Stdout = tty
	cmd.Stderr = tty
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
	}

	t.StdFile = pty // Stdin + Stdout + Stderr
	t.reader = bufio.NewReader(t.StdFile)
	termios.Tcgetattr(t.StdFile.Fd(), &t.Termios)
	if t.width > 0 && t.height > 0 {
		t.setsize()
	}
	if t.ospeed > 0 && t.ispeed > 0 {
		t.setspeed()
	}

	err = cmd.Start()
	if err != nil {
		pty.Close()
	}
	return err
}

// Modifies termios for raw mode
func (t *Terminal) setRawMode() {
	t.Termios.Iflag &^= unix.ISTRIP | unix.INLCR | unix.ICRNL | unix.IGNCR | unix.IXOFF
	t.Termios.Lflag &^= unix.ECHO | unix.ICANON
	t.Termios.Cc[unix.VMIN] = 1
	t.Termios.Cc[unix.VTIME] = 0
	termios.Tcsetattr(t.StdFile.Fd(), termios.TCSANOW, &t.Termios)
}

func (t *Terminal) SetType(terminalType string) {
	t.Type = terminalType
	t.EnvChan <- append(os.Environ(), "TERM="+terminalType)
}

func (t *Terminal) GetSize() (height int, width int, err error) {
	ws, err := unix.IoctlGetWinsize(int(t.StdFile.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Row), int(ws.Col), nil
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
	err := unix.IoctlSetWinsize(int(t.StdFile.Fd()), unix.TIOCSWINSZ, &unix.Winsize{
		Row: t.height,
		Col: t.width,
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
	// To support both 32bit and 64bit
	ospeedRv := reflect.ValueOf(&t.Termios.Ospeed)
	ispeedRv := reflect.ValueOf(&t.Termios.Ispeed)
	ospeedRv.Elem().SetUint(uint64(t.ispeed))
	ispeedRv.Elem().SetUint(uint64(t.ispeed))
	// Set new termios to StdFile
	termios.Tcsetattr(t.StdFile.Fd(), termios.TCSANOW, &t.Termios)
}

func (t *Terminal) Close() error {
	if t.StdFile != nil {
		return t.StdFile.Close()
	}
	return nil
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

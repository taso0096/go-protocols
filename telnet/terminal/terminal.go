package terminal

import (
	"os"

	"github.com/mattn/go-tty"
	"github.com/pkg/term/termios"
	"golang.org/x/sys/unix"
)

func NewFromTty(tty *tty.TTY) Terminal {
	t := Terminal{
		tty:     tty,
		StdFile: tty.Input(),
	}
	termios.Tcgetattr(uintptr(tty.Input().Fd()), &t.Termios)
	t.Type = os.Getenv("TERM")
	if len(t.Type) == 0 {
		t.Type = "VT100"
	}

	return t
}

type Terminal struct {
	tty     *tty.TTY
	StdFile *os.File
	Termios unix.Termios
	Type    string
}

func (t *Terminal) GetSize() (int, int, error) {
	return t.tty.Size()
}

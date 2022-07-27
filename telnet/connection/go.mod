module telnet/connection

go 1.18

replace (
	telnet/command => ../command
	telnet/option => ../option
	telnet/terminal => ../terminal
)

require (
	telnet/command v0.0.0-00010101000000-000000000000
	telnet/option v0.0.0-00010101000000-000000000000
	telnet/terminal v0.0.0-00010101000000-000000000000
)

require (
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mattn/go-tty v0.0.4 // indirect
	github.com/pkg/term v1.1.0 // indirect
	golang.org/x/sys v0.0.0-20220727055044-e65921a090b8 // indirect
)

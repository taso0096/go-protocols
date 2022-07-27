module telnet/client

go 1.18

replace (
	telnet/command => ../command
	telnet/connection => ../connection
	telnet/option => ../option
)

require (
	github.com/mattn/go-tty v0.0.4
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1
	telnet/command v0.0.0-00010101000000-000000000000
	telnet/connection v0.0.0-00010101000000-000000000000
	telnet/option v0.0.0-00010101000000-000000000000
)

require github.com/mattn/go-isatty v0.0.10 // indirect

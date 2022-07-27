module telnet

go 1.18

replace (
	telnet/client => ./client
	telnet/command => ./command
	telnet/connection => ./connection
	telnet/option => ./option
	telnet/server => ./server
)

require (
	telnet/client v0.0.0-00010101000000-000000000000
	telnet/server v0.0.0-00010101000000-000000000000
)

require (
	github.com/creack/pty v1.1.18 // indirect
	github.com/mattn/go-isatty v0.0.10 // indirect
	github.com/mattn/go-tty v0.0.4 // indirect
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 // indirect
	telnet/command v0.0.0-00010101000000-000000000000 // indirect
	telnet/connection v0.0.0-00010101000000-000000000000 // indirect
	telnet/option v0.0.0-00010101000000-000000000000 // indirect
)

module telnet/client

go 1.18

replace (
	telnet/command => ../command
	telnet/connection => ../connection
)

require (
	github.com/mattn/go-tty v0.0.4
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d
	telnet/command v0.0.0-00010101000000-000000000000
	telnet/connection v0.0.0-00010101000000-000000000000
)

require (
	github.com/mattn/go-isatty v0.0.10 // indirect
	golang.org/x/sys v0.0.0-20210615035016-665e8c7367d1 // indirect
	golang.org/x/term v0.0.0-20201126162022-7de9c90e9dd1 // indirect
)

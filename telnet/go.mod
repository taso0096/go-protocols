module telnet

go 1.18

replace (
	telnet/client => ./client
	telnet/command => ./command
	telnet/connection => ./connection
	telnet/option => ./option
	telnet/server => ./server
	telnet/terminal => ./terminal
)

require (
	telnet/client v0.0.0-00010101000000-000000000000
	telnet/server v0.0.0-00010101000000-000000000000
)

require (
	github.com/pkg/term v1.1.0 // indirect
	golang.org/x/sys v0.0.0-20220727055044-e65921a090b8 // indirect
	telnet/command v0.0.0-00010101000000-000000000000 // indirect
	telnet/connection v0.0.0-00010101000000-000000000000 // indirect
	telnet/option v0.0.0-00010101000000-000000000000 // indirect
	telnet/terminal v0.0.0-00010101000000-000000000000 // indirect
)

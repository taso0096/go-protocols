module telnet/server

go 1.18

replace (
	telnet/command => ../command
	telnet/connection => ../connection
	telnet/option => ../option
)

require telnet/connection v0.0.0-00010101000000-000000000000

require (
	telnet/command v0.0.0-00010101000000-000000000000
	telnet/option v0.0.0-00010101000000-000000000000
)

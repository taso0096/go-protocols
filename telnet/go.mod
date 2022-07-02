module telnet

go 1.18

replace (
	telnet/client => ./client
	telnet/connection => ./connection
)

require telnet/client v0.0.0-00010101000000-000000000000

require telnet/connection v0.0.0-00010101000000-000000000000 // indirect

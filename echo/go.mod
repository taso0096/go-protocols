module main.go

go 1.18

replace (
	client => ./client
	connection => ./connection
	server => ./server
)

require (
	client v0.0.0-00010101000000-000000000000
	server v0.0.0-00010101000000-000000000000
)

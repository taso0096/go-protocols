module main.go

go 1.16

replace (
	client => ./client
	server => ./server
)

require (
	client v0.0.0-00010101000000-000000000000
	server v0.0.0-00010101000000-000000000000
)
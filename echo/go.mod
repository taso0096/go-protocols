module echo

go 1.18

replace (
	echo/client => ./client
	echo/connection => ./connection
	echo/server => ./server
)

require (
	echo/client v0.0.0-00010101000000-000000000000
	echo/server v0.0.0-00010101000000-000000000000
)

require echo/connection v0.0.0-00010101000000-000000000000 // indirect

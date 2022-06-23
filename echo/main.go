package main

import (
	"echo/client"
	"echo/server"
	"flag"
)

func main() {
	ip := flag.String("ip", "0.0.0.0", "IP")
	port := flag.Int("port", 7, "Port")
	isServerMode := flag.Bool("s", false, "Start echo server (default echo client)")
	flag.Parse()

	if *isServerMode {
		server.Run(*ip, *port)
	} else {
		client.Run(*ip, *port)
	}
}

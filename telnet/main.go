package main

import (
	"flag"
	"telnet/client"
	"telnet/server"
)

func main() {
	ip := flag.String("ip", "0.0.0.0", "IP")
	port := flag.Int("port", 23, "Port")
	isServerMode := flag.Bool("s", false, "Start telnet server (default telnet client)")
	flag.Parse()

	if *isServerMode {
		server.Run(*ip, *port)
	} else {
		client.Run(*ip, *port)
	}
}

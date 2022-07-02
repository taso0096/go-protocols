package main

import (
	"flag"
	"fmt"
	"telnet/client"
)

func main() {
	ip := flag.String("ip", "0.0.0.0", "IP")
	port := flag.Int("port", 23, "Port")
	isServerMode := flag.Bool("s", false, "Start telnet server (default telnet client)")
	flag.Parse()

	if *isServerMode {
		fmt.Println("Server is not yet implemented.")
	} else {
		client.Run(*ip, *port)
	}
}

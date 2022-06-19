package main

import (
	"client"
	"flag"
	"io"
	"log"
	"server"
)

func startClient(ip string, port int) {
	client := client.Init(ip, port)

	err := client.Call()
	if err != nil {
		log.Fatal("Call Error:", err)
	}
	defer client.Conn.Close()

	err = client.Write("Hi\x00")
	if err != nil {
		log.Fatal("Write Error:", err)
	}

	message, err := client.Read()
	if err != nil {
		log.Fatal("Read Error:", err)
	}
	log.Println(message)
}

func startServer(ip string, port int) {
	server := server.Init(ip, port)

	for {
		err := server.Listen()
		if err != nil {
			log.Fatal("Listen Error:", err)
		}
		defer server.Conn.Close()

		message, err := server.Read()
		if err != nil && err != io.EOF {
			log.Fatal("Read Error:", err)
		}
		log.Println(message)

		err = server.Write(message)
		if err != nil {
			log.Fatal("Write Error:", err)
		}
	}
}

func main() {
	ip := flag.String("ip", "0.0.0.0", "IP")
	port := flag.Int("port", 7, "Port")
	isServerMode := flag.Bool("s", false, "Start echo server (default echo client)")
	flag.Parse()

	if *isServerMode {
		startServer(*ip, *port)
	} else {
		startClient(*ip, *port)
	}
}

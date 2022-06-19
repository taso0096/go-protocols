package main

import (
	"client"
	"flag"
	"io"
	"log"
	"server"
)

func startClient() {
	client := client.Client{
		IP:   "0.0.0.0",
		Port: 10007,
	}

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

func startServer() {
	server := server.Server{
		IP:   "0.0.0.0",
		Port: 10007,
	}

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
	isServerMode := flag.Bool("s", false, "Start echo server (default: echo client)")
	flag.Parse()

	if *isServerMode {
		startServer()
	} else {
		startClient()
	}
}

package main

import (
	"client"
	"log"
)

func startClient() {
	client := client.Client{
		IP:   "0.0.0.0",
		Port: 10007,
	}

	err := client.Call()
	if err != nil {
		log.Fatal(err)
	}
	defer client.Conn.Close()

	client.Write("Hi\x00")
	if err != nil {
		log.Fatal(err)
	}

	message, err := client.Read()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(message)
}

func main() {
	startClient()
}

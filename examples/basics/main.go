package main

import (
	"net"

	simpleSTUN "github.com/simpleNetworking/go-simplestun"
)

func main() {
	//start a udp listener on port 0
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ip, port, err := simpleSTUN.GetPublicIPPort(conn, nil)
	if err != nil {
		panic(err)
	}

	println("Public IP:", ip)
	println("Public Port:", port)
}

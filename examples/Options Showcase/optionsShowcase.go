package main

import (
	"net"

	"simpleSTUN"
)

func main() {
	//start a udp listener on port 0
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ip, port, err := simpleSTUN.GetPublicIPPort(conn, &simpleSTUN.Options{   // You CAN do this but that doesn't mean you should
    // StunServerName : "stun.nextcloud.com"     
		StunServerIP:   net.IPv4(159, 69, 191, 124), // Use either name OR IP, don't try to use both. 
		StunServerPort: 3478,
	})
	if err != nil {
		panic(err)
	}

	println("Public IP:", ip)
	println("Public Port:", port)
}

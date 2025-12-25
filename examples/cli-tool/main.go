package main

import (
	"flag"
	"net"
	"strconv"
	"strings"

	simpleSTUN "github.com/simpleNetworking/go-simplestun"
)

func main() {
	//parse command line flags
	var stunServer string = "stun.l.google.com"
	var stunPort int = 19302
	localPort := flag.Int("port", 0, "Local port to bind to")
	servername := flag.String("server", "stun.l.google.com:19302", "STUN server to use")
	flag.Parse()
	if *servername != "" {
		h := strings.Split(*servername, ":")
		stunServer = h[0]
		stunPort, _ = strconv.Atoi(h[1])
	}
	//start a udp listener on specified port
	conn, err := net.ListenPacket("udp", ":"+strconv.Itoa(*localPort))
	if err != nil {
		panic(err)
	}
	defer conn.Close() //ensure we close the listener when done

	// And ofc query the STUN server for our public IP/port
	ip, port, err := simpleSTUN.GetPublicIPPort(conn, &simpleSTUN.Options{
		StunServerName: stunServer,
		StunServerPort: stunPort,
	})
	if err != nil {
		panic(err)
	}

	println("Public IP:", ip)
	if *localPort != 0 {
		println("Public Port:", port)
	} else {
		println("Public Port (random local port):", port)
	}
}

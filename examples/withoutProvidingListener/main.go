package main

import (
	simpleSTUN "github.com/simpleNetworking/go-simplestun"
)

func main() {
	ip, port, err := simpleSTUN.GetPublicIPPort(nil, &simpleSTUN.Options{
		LocalPort: 2157,
	})
	if err != nil {
		panic(err)
	}

	println("Public IP:", ip)
	println("Public Port:", port)
}

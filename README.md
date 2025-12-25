# go-simplestun

An overly simplistic **STUN** (Session Traversal Utilities for NAT) library in Go to discover your public IP address and port from behind NAT.

The library provides you with only ONE function to get the job done, as every STUN library should.
(RFC5489 and RFC8489 compatible)

## Features

- Minimal, dependency-free STUN client.
- Retrieves the public (mapped) IP and port as seen by the STUN server.
- Pure standard library usage.
- Multiple usage examples provided.

## Installation

```bash
go get github.com/SimpleNetworking/go-simplestun
```

# Usage
Import the package:

```bash
import "github.com/SimpleNetworking/go-simplestun"
```
The library exposes a single primary function for most use cases, as every STUN library **should**.

```go
import (
	"net"
	"github.com/SimpleNetworking/go-simplestun"
)

func main() {
	//start a udp listener on port 0 (or any other port of course)
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
  // Of course, you don't HAVE to do this if you already have a listener you want to use

	ip, port, err := simpleSTUN.GetPublicIPPort(conn, nil) //pass the listener to this function and done
	if err != nil {
		panic(err)
	}

	println("Public IP:", ip)
	println("Public Port:", port)
  // ...
  // And now you can share these with peers. Or do whatever you want, for that matter
}
```
Of course you don't HAVE to create a listener yourself, you may provide the function with a port number and it will create a listener on that specific port for you. Alternatively, you may not pass anything at all, and the function will create a random port on its own. Do that only if **only** need the IP and don't care about the port number.

For more advanced usage, different options, or custom listeners, check the comprehensive examples in the examples/ folder.

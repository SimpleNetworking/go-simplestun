package simpleSTUN

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

const defaultSTUNServer = "stun.l.google.com:19302"

type Options struct {
	LocalPort      int
	StunServerName string // Plz don't try to use both ServerName and ServerIP
	StunServerIP   net.IP
	StunServerPort int
}

type response struct {
	data   []byte
	length int
	addr   net.Addr
	err    error
}

func GetPublicIPPort(conn net.PacketConn, opts *Options) (string, int, error) {
	connProvided := conn != nil
	portProvided := opts != nil && opts.LocalPort != 0

	if opts != nil && opts.StunServerName != "" && opts.StunServerIP != nil {
		return "", 0, fmt.Errorf("giving both name and IP? I guess this is what you wanted to see? an error?")
	}

	// --- 2. HANDLE conn == nil CASE (Return Local IP) ---
	if conn == nil {
		// Use net.Dial to determine the local IP assigned by the OS, without sending data.
		conn, _ = net.ListenPacket("udp", fmt.Sprintf(":%d", opts.LocalPort))
		defer conn.Close()
	}

	// --- 3. Determine STUN Server Address ---
	serverAddrStr := defaultSTUNServer
	if opts != nil {
		if opts.StunServerName != "" {
			serverAddrStr = fmt.Sprintf("%s:%d", opts.StunServerName, opts.StunServerPort)
		} else if opts.StunServerIP != nil {
			serverAddrStr = fmt.Sprintf("%s:%d", opts.StunServerIP.String(), opts.StunServerPort)
		}
	}

	serverAddr, err := net.ResolveUDPAddr("udp", serverAddrStr)
	if err != nil {
		return "", 0, fmt.Errorf("failed to resolve STUN server address %s: %w", serverAddrStr, err)
	}

	// --- 4. Prepare STUN Binding Request ---
	var requestHeader = [20]byte{
		0x00, 0x01, // Message Type: Binding Request
		0x00, 0x00, // Message Length: 0
		0x21, 0x12, 0xA4, 0x42, // Magic Cookie
	}
	if _, err := rand.Read(requestHeader[8:20]); err != nil {
		return "", 0, fmt.Errorf("could not generate Transaction ID: %w", err)
	}
	transactionID := requestHeader[8:20]

	responseChan := make(chan response, 1)
	readTimeout := 20 * time.Second // 5 seconds is standard for STUN

	if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return "", 0, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// --- 5. Start Listener Goroutine and Send Request ---
	go startListener(conn, responseChan)

	bytesWritten, err := conn.WriteTo(requestHeader[:], serverAddr)
	if bytesWritten != len(requestHeader) || err != nil {
		return "", 0, fmt.Errorf("failed to send STUN request: %w", err)
	}

	// --- 6. Wait for Response or Timeout ---
	var res response
	select {
	case res = <-responseChan:
		// Error will be handled below
	case <-time.After(readTimeout):
		return "", 0, fmt.Errorf("STUN request timed out after %v", readTimeout)
	}
	if res.err != nil {
		if netErr, ok := res.err.(net.Error); ok && netErr.Timeout() {
			return "", 0, fmt.Errorf("STUN read failed due to connection timeout")
		}
		return "", 0, fmt.Errorf("failed to read STUN response: %w", res.err)
	}

	// --- 7. DECODING LOGIC ---
	if res.length < 20 {
		return "", 0, fmt.Errorf("STUN response too short (%d bytes)", res.length)
	}

	// Validate Message Type (0x0101) and Transaction ID
	if binary.BigEndian.Uint16(res.data[0:2]) != 0x0101 {
		return "", 0, fmt.Errorf("received unexpected STUN message type: %x", binary.BigEndian.Uint16(res.data[0:2]))
	}
	if !bytes.Equal(res.data[8:20], transactionID) {
		return "", 0, fmt.Errorf("STUN transaction ID mismatch")
	}

	// Iterate and Find XOR-MAPPED-ADDRESS (Type 0x0020)
	attrOffset := 20
	for attrOffset < res.length {
		attrType := binary.BigEndian.Uint16(res.data[attrOffset : attrOffset+2])
		attrLen := binary.BigEndian.Uint16(res.data[attrOffset+2 : attrOffset+4])

		if attrType == 0x0020 {
			value := res.data[attrOffset+4 : attrOffset+4+int(attrLen)]
			publicIP, publicPort, err := decodeXorMappedAddress(value) // Transaction ID is now baked into the decoding logic
			if err != nil {
				return "", 0, fmt.Errorf("decoding error: %w", err)
			}
			if !connProvided && !portProvided {
				return publicIP.String(), 0, nil
			}
			return publicIP.String(), publicPort, nil
		}

		// Advance to the next attribute, respecting 4-byte padding
		attrOffset += 4 + int(attrLen)
		if pad := attrOffset % 4; pad != 0 {
			attrOffset += 4 - pad
		}
	}

	return "", 0, fmt.Errorf("STUN response did not contain XOR-MAPPED-ADDRESS")
}

func startListener(conn net.PacketConn, responseChan chan response) {
	buf := make([]byte, 1024)
	n, addr, err := conn.ReadFrom(buf)

	if err != nil {
		responseChan <- response{err: err}
		return
	}
	responseChan <- response{data: buf[:n], addr: addr, length: n}

}

// decodeXorMappedAddress decodes the XOR-MAPPED-ADDRESS attribute value.
func decodeXorMappedAddress(value []byte) (net.IP, int, error) {

	if len(value) < 8 {
		return nil, 0, fmt.Errorf("invalid XOR-MAPPED-ADDRESS length")
	}

	// Magic Cookie: 0x2112A442
	const magicCookie = 0x2112A442

	family := value[1]
	xorPort := binary.BigEndian.Uint16(value[2:4])
	decodedPort := xorPort ^ uint16(magicCookie>>16) // Port XORed with 0x2112

	var ip net.IP
	switch family {
	case 0x01: // IPv4
		xorIP := binary.BigEndian.Uint32(value[4:8])
		decodedIP := xorIP ^ magicCookie

		ipBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(ipBytes, decodedIP)
		ip = net.IP(ipBytes)

	case 0x02: // IPv6
		return nil, 0, fmt.Errorf("IPv6 decoding not supported by simpleSTUN")

	default:
		return nil, 0, fmt.Errorf("unknown address family: %d", family)
	}

	return ip, int(decodedPort), nil
}

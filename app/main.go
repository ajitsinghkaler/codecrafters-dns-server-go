package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

// Ensures gofmt doesn't remove the "net" import in stage 1 (feel free to remove this!)

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	// TODO: Uncomment the code below to pass the first stage
	//
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		// Create an empty response
		response := []byte{}

		writeBuf := new(bytes.Buffer)

		var num uint16 = 1234

		err = binary.Write(writeBuf, binary.BigEndian, num)

		if err != nil {
			fmt.Println("Error converting id to bytes1")
		}

		var headerVal uint8 = 1 << 7

		err = binary.Write(writeBuf, binary.BigEndian, headerVal)

		if err != nil {
			fmt.Println("Error converting id to bytes2")
		}

		headerVal = 0

		err = binary.Write(writeBuf, binary.BigEndian, headerVal)

		if err != nil {
			fmt.Println("Error converting id to bytes3")
		}

		var qdcount uint16 = 0

		err = binary.Write(writeBuf, binary.BigEndian, qdcount)

		if err != nil {
			fmt.Println("Error converting id to bytes4")
		}

		var ancount uint16 = 0

		err = binary.Write(writeBuf, binary.BigEndian, ancount)

		if err != nil {
			fmt.Println("Error converting id to bytes5")
		}

		var nscount uint16 = 0

		err = binary.Write(writeBuf, binary.BigEndian, nscount)

		if err != nil {
			fmt.Println("Error converting id to bytes6")
		}
		var arcount uint16 = 0

		err = binary.Write(writeBuf, binary.BigEndian, arcount)

		if err != nil {
			fmt.Println("Error converting id to bytes7")
		}

		response = append(response, writeBuf.Bytes()...)
		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}

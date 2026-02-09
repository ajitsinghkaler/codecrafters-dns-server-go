package main

import (
	"encoding/binary"
	"fmt"
	"net"
)

type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

func main() {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			break
		}

		if size < 12 {
			continue
		}

		requestID := binary.BigEndian.Uint16(buf[0:2])

		header := DNSHeader{
			ID:      requestID,
			Flags:   createFlags(1, 0, 0, 0, 0, 0, 0, 0),
			QDCount: 0,
			ANCount: 0,
			NSCount: 0,
			ARCount: 0,
		}

		response := make([]byte, 12)
		binary.BigEndian.PutUint16(response[0:2], header.ID)
		binary.BigEndian.PutUint16(response[2:4], header.Flags)
		binary.BigEndian.PutUint16(response[4:6], header.QDCount)
		binary.BigEndian.PutUint16(response[6:8], header.ANCount)
		binary.BigEndian.PutUint16(response[8:10], header.NSCount)
		binary.BigEndian.PutUint16(response[10:12], header.ARCount)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

func createFlags(qr, opcode, aa, tc, rd, ra, z, rcode uint16) uint16 {
	return (qr << 15) | (opcode << 11) | (aa << 10) | (tc << 9) | (rd << 8) | (ra << 7) | (z << 4) | rcode
}

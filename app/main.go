package main

import (
	"encoding/binary"
	"net"
)

func main() {
	udpAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	udpConn, _ := net.ListenUDP("udp", udpAddr)
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
		qdCount := binary.BigEndian.Uint16(buf[4:6])
		byte2 := buf[2]
		opcode := (byte2 >> 3) & 0x0F
		rd := byte2 & 0x01
		var rcode byte = 0
		if opcode != 0 {
			rcode = 4
		}

		flags := uint16(1)<<15 | uint16(opcode)<<11 | uint16(rd)<<8 | uint16(rcode)

		response := make([]byte, 12)
		binary.BigEndian.PutUint16(response[0:2], requestID)
		binary.BigEndian.PutUint16(response[2:4], flags)
		binary.BigEndian.PutUint16(response[4:6], qdCount)
		binary.BigEndian.PutUint16(response[6:8], qdCount)

		currentOffset := 12
		var names [][]byte

		for i := 0; i < int(qdCount); i++ {
			name, nextOffset := parseName(buf, currentOffset)
			names = append(names, name)

			response = append(response, name...)
			qTypeClass := make([]byte, 4)
			binary.BigEndian.PutUint16(qTypeClass[0:2], 1)
			binary.BigEndian.PutUint16(qTypeClass[2:4], 1)
			response = append(response, qTypeClass...)

			currentOffset = nextOffset + 4
		}

		for _, name := range names {
			answer := createAnswer(name, "8.8.8.8")
			response = append(response, answer...)
		}

		udpConn.WriteToUDP(response, source)
	}
}

func parseName(packet []byte, offset int) ([]byte, int) {
	var name []byte
	ptrFound := false
	nextOffsetAfterName := 0
	curr := offset

	for {
		if curr >= len(packet) {
			break
		}
		b := packet[curr]
		if b == 0 {
			name = append(name, 0)
			if !ptrFound {
				nextOffsetAfterName = curr + 1
			}
			break
		}

		if b&0xC0 == 0xC0 {
			if !ptrFound {
				nextOffsetAfterName = curr + 2
				ptrFound = true
			}
			ptr := int(binary.BigEndian.Uint16(packet[curr:curr+2]) & 0x3FFF)
			curr = ptr
		} else {
			length := int(b)
			name = append(name, packet[curr:curr+length+1]...)
			curr += length + 1
		}
	}

	return name, nextOffsetAfterName
}

func createAnswer(name []byte, ip string) []byte {
	res := append([]byte{}, name...)

	meta := make([]byte, 10)
	binary.BigEndian.PutUint16(meta[0:2], 1)
	binary.BigEndian.PutUint16(meta[2:4], 1)
	binary.BigEndian.PutUint32(meta[4:8], 60)
	binary.BigEndian.PutUint16(meta[8:10], 4)
	res = append(res, meta...)

	parsedIP := net.ParseIP(ip).To4()
	res = append(res, parsedIP...)

	return res
}

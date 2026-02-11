package main

import (
	"encoding/binary"
	"log"
	"net"
)

const (
	DNSPort       = "127.0.0.1:2053"
	HeaderSize    = 12
	TypeA         = 1
	ClassInternet = 1
)

type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
}

type DNSQuestion struct {
	Name  []byte
	Type  uint16
	Class uint16
}

func main() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{Port: 2053})
	if err != nil {
		log.Fatalf("Socket error: %v", err)
	}
	defer conn.Close()

	buffer := make([]byte, 512)
	for {
		size, remoteAddr, _ := conn.ReadFromUDP(buffer)
		if size < HeaderSize {
			continue
		}

		response := processPacket(buffer[:size])
		conn.WriteToUDP(response, remoteAddr)
	}
}

func processPacket(rawRequest []byte) []byte {
	header := parseHeader(rawRequest)
	header.Flags = buildResponseFlags(rawRequest[2])
	header.ANCount = header.QDCount

	response := serializeHeader(header)
	questions, _ := parseAllQuestions(rawRequest, int(header.QDCount))

	for _, q := range questions {
		response = append(response, encodeQuestion(q)...)
	}

	for _, q := range questions {
		answer := buildARecord(q.Name, "8.8.8.8")
		response = append(response, answer...)
	}

	return response
}

func parseHeader(data []byte) DNSHeader {
	return DNSHeader{
		ID:      binary.BigEndian.Uint16(data[0:2]),
		QDCount: binary.BigEndian.Uint16(data[4:6]),
	}
}

func buildResponseFlags(requestByte2 byte) uint16 {
	opcode := (requestByte2 >> 3) & 0x0F
	rd := requestByte2 & 0x01
	rcode := uint16(0)
	if opcode != 0 {
		rcode = 4
	}
	return uint16(1)<<15 | uint16(opcode)<<11 | uint16(rd)<<8 | rcode
}

func serializeHeader(h DNSHeader) []byte {
	buf := make([]byte, HeaderSize)
	binary.BigEndian.PutUint16(buf[0:2], h.ID)
	binary.BigEndian.PutUint16(buf[2:4], h.Flags)
	binary.BigEndian.PutUint16(buf[4:6], h.QDCount)
	binary.BigEndian.PutUint16(buf[6:8], h.ANCount)
	return buf
}

func parseAllQuestions(data []byte, count int) ([]DNSQuestion, int) {
	offset := HeaderSize
	questions := make([]DNSQuestion, 0, count)

	for i := 0; i < count; i++ {
		name, nextOffset := parseName(data, offset)
		q := DNSQuestion{
			Name:  name,
			Type:  binary.BigEndian.Uint16(data[nextOffset : nextOffset+2]),
			Class: binary.BigEndian.Uint16(data[nextOffset+2 : nextOffset+4]),
		}
		questions = append(questions, q)
		offset = nextOffset + 4
	}
	return questions, offset
}

func encodeQuestion(q DNSQuestion) []byte {
	buf := append([]byte{}, q.Name...)
	meta := make([]byte, 4)
	binary.BigEndian.PutUint16(meta[0:2], q.Type)
	binary.BigEndian.PutUint16(meta[2:4], q.Class)
	return append(buf, meta...)
}

func buildARecord(name []byte, ip string) []byte {
	record := append([]byte{}, name...)

	meta := make([]byte, 10)
	binary.BigEndian.PutUint16(meta[0:2], TypeA)
	binary.BigEndian.PutUint16(meta[2:4], ClassInternet)
	binary.BigEndian.PutUint32(meta[4:8], 60)
	binary.BigEndian.PutUint16(meta[8:10], 4)

	record = append(record, meta...)
	record = append(record, net.ParseIP(ip).To4()...)
	return record
}

func parseName(packet []byte, offset int) ([]byte, int) {
	curr := offset
	var name []byte
	ptrFound := false
	endOffset := 0

	for {
		b := packet[curr]
		if b == 0 {
			name = append(name, 0)
			if !ptrFound {
				endOffset = curr + 1
			}
			break
		}
		if b&0xC0 == 0xC0 {
			if !ptrFound {
				endOffset = curr + 2
				ptrFound = true
			}
			curr = int(binary.BigEndian.Uint16(packet[curr:curr+2]) & 0x3FFF)
		} else {
			length := int(b)
			name = append(name, packet[curr:curr+length+1]...)
			curr += length + 1
		}
	}
	return name, endOffset
}

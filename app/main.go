package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type DNSHeader struct {
	ID      uint16
	Flags   uint16
	QDCount uint16
	ANCount uint16
	NSCount uint16
	ARCount uint16
}

type DNSQuestion struct {
	DName []byte
	Type  uint16
	Class uint16
}

type DNSAnswer struct {
	DName    []byte
	Type     uint16
	Class    uint16
	TTL      uint32
	RDLength uint16
	Data     []byte
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

		response := createHeader(buf)
		response = append(response, createQuestion()...)
		response = append(response, createAnswer()...)

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

func createFlags(qr, opcode, aa, tc, rd, ra, z, rcode uint16) uint16 {
	return (qr << 15) | (opcode << 11) | (aa << 10) | (tc << 9) | (rd << 8) | (ra << 7) | (z << 4) | rcode
}

func createHeader(buf []byte) []byte {

	flag1byte := buf[2]

	opcode := uint16((flag1byte >> 3) & 0x0F)

	rd := uint16(flag1byte & 0x01)
	var rcode uint16 = 0
	if opcode != 0 {
		rcode = 4
	}

	header := DNSHeader{
		ID:      binary.BigEndian.Uint16(buf[0:2]),
		Flags:   createFlags(1, opcode, 0, 0, rd, 0, 0, rcode),
		QDCount: 1,
		ANCount: 1,
		NSCount: 0,
		ARCount: 0,
	}

	headerRes := make([]byte, 12)
	binary.BigEndian.PutUint16(headerRes[0:2], header.ID)
	binary.BigEndian.PutUint16(headerRes[2:4], header.Flags)
	binary.BigEndian.PutUint16(headerRes[4:6], header.QDCount)
	binary.BigEndian.PutUint16(headerRes[6:8], header.ANCount)
	binary.BigEndian.PutUint16(headerRes[8:10], header.NSCount)
	binary.BigEndian.PutUint16(headerRes[10:12], header.ARCount)
	return headerRes
}

func createQuestion() []byte {
	var questionRes []byte
	question := DNSQuestion{
		DName: convertStringToUrlBytes("codecrafters.io"),
		Type:  1,
		Class: 1,
	}

	questionRes = append(questionRes, question.DName...)
	typeBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBuf, question.Type)
	questionRes = append(questionRes, typeBuf...)

	classBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(classBuf, question.Class)
	questionRes = append(questionRes, classBuf...)

	return questionRes
}

func createAnswer() []byte {
	var answerRes []byte
	answer := DNSAnswer{
		DName:    convertStringToUrlBytes("codecrafters.io"),
		Type:     1,
		Class:    1,
		TTL:      60,
		RDLength: 4,
		Data:     convertIPtoByte("8.8.8.8"),
	}

	answerRes = append(answerRes, answer.DName...)
	typeBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBuf, answer.Type)
	answerRes = append(answerRes, typeBuf...)

	classBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(classBuf, answer.Class)
	answerRes = append(answerRes, classBuf...)

	TTLbuf := make([]byte, 4)
	binary.BigEndian.PutUint32(TTLbuf, answer.TTL)
	answerRes = append(answerRes, TTLbuf...)

	rdLengthBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(rdLengthBuf, answer.RDLength)
	answerRes = append(answerRes, rdLengthBuf...)

	answerRes = append(answerRes, answer.Data...)

	return answerRes
}

func convertStringToUrlBytes(url string) []byte {
	var urlBytes []byte
	urlParts := strings.Split(url, ".")

	for _, part := range urlParts {
		urlBytes = append(urlBytes, byte(len(part)))
		urlPartBuf := []byte(part)
		urlBytes = append(urlBytes, urlPartBuf...)
	}
	urlBytes = append(urlBytes, 0)
	return urlBytes
}

func convertIPtoByte(ip string) []byte {
	ipParts := strings.Split(ip, ".")
	var ipBytes []byte

	for _, part := range ipParts {
		num, err := strconv.Atoi(part)
		if err != nil {
			num = 0
		}
		ipBytes = append(ipBytes, byte(num))
	}

	return ipBytes
}

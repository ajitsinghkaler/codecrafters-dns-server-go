package main

import (
	"encoding/binary"
	"flag"
	"log"
	"net"
)

const (
	ListenAddr = "127.0.0.1:2053"
	HeaderSize = 12
)

type DNSHeader struct {
	PacketID      uint16
	Flags         uint16
	QuestionCount uint16
	AnswerCount   uint16
}

type DNSQuestion struct {
	Name  []byte
	Type  uint16
	Class uint16
}

func main() {
	resolverAddress := flag.String("resolver", "", "")
	flag.Parse()

	connection, err := net.ListenUDP("udp", &net.UDPAddr{Port: 2053})
	if err != nil {
		log.Fatalf("Socket error: %v", err)
	}
	defer connection.Close()

	readBuffer := make([]byte, 512)
	for {
		bytesRead, remoteAddress, _ := connection.ReadFromUDP(readBuffer)
		if bytesRead < HeaderSize {
			continue
		}

		responsePacket := handleRequest(readBuffer[:bytesRead], *resolverAddress)
		connection.WriteToUDP(responsePacket, remoteAddress)
	}
}

func handleRequest(rawRequest []byte, resolver string) []byte {
	header := parseHeader(rawRequest)
	questions, _ := parseAllQuestions(rawRequest, int(header.QuestionCount))

	opcode := (rawRequest[2] >> 3) & 0x0F
	recursionDesired := uint16(rawRequest[2] & 0x01)
	responseCode := uint16(0)
	if opcode != 0 {
		responseCode = 4
	}

	var allAnswers []byte
	if responseCode == 0 {
		for _, question := range questions {
			answerPacket := forwardQuery(question, header.PacketID, resolver)
			if len(answerPacket) > HeaderSize {
				upstreamAnswerCount := binary.BigEndian.Uint16(answerPacket[6:8])
				_, questionNextOffset := parseName(answerPacket, HeaderSize)

				answerData := answerPacket[questionNextOffset+4:]
				if upstreamAnswerCount > 0 {
					allAnswers = append(allAnswers, answerData...)
				}
			}
		}
	}

	header.Flags = uint16(1)<<15 | uint16(opcode)<<11 | recursionDesired<<8 | responseCode
	header.AnswerCount = uint16(len(questions))

	response := serializeHeader(header)
	for _, question := range questions {
		response = append(response, encodeQuestion(question)...)
	}
	response = append(response, allAnswers...)

	return response
}

func forwardQuery(question DNSQuestion, packetID uint16, resolver string) []byte {
	destination, _ := net.ResolveUDPAddr("udp", resolver)
	connection, err := net.DialUDP("udp", nil, destination)
	if err != nil {
		return nil
	}
	defer connection.Close()

	header := DNSHeader{
		PacketID:      packetID,
		QuestionCount: 1,
		Flags:         0x0100,
	}

	packet := serializeHeader(header)
	packet = append(packet, encodeQuestion(question)...)

	connection.Write(packet)
	buffer := make([]byte, 512)
	bytesReceived, _ := connection.Read(buffer)
	return buffer[:bytesReceived]
}

func parseHeader(data []byte) DNSHeader {
	return DNSHeader{
		PacketID:      binary.BigEndian.Uint16(data[0:2]),
		Flags:         binary.BigEndian.Uint16(data[2:4]),
		QuestionCount: binary.BigEndian.Uint16(data[4:6]),
	}
}

func serializeHeader(header DNSHeader) []byte {
	buffer := make([]byte, HeaderSize)
	binary.BigEndian.PutUint16(buffer[0:2], header.PacketID)
	binary.BigEndian.PutUint16(buffer[2:4], header.Flags)
	binary.BigEndian.PutUint16(buffer[4:6], header.QuestionCount)
	binary.BigEndian.PutUint16(buffer[6:8], header.AnswerCount)
	return buffer
}

func parseAllQuestions(data []byte, count int) ([]DNSQuestion, int) {
	currentOffset := HeaderSize
	questions := make([]DNSQuestion, 0, count)

	for i := 0; i < count; i++ {
		name, nextOffset := parseName(data, currentOffset)
		question := DNSQuestion{
			Name:  name,
			Type:  binary.BigEndian.Uint16(data[nextOffset : nextOffset+2]),
			Class: binary.BigEndian.Uint16(data[nextOffset+2 : nextOffset+4]),
		}
		questions = append(questions, question)
		currentOffset = nextOffset + 4
	}
	return questions, currentOffset
}

func encodeQuestion(question DNSQuestion) []byte {
	buffer := append([]byte{}, question.Name...)
	metadata := make([]byte, 4)
	binary.BigEndian.PutUint16(metadata[0:2], question.Type)
	binary.BigEndian.PutUint16(metadata[2:4], question.Class)
	return append(buffer, metadata...)
}

func parseName(packet []byte, offset int) ([]byte, int) {
	currentPosition := offset
	var name []byte
	pointerFound := false
	endOffset := 0

	for {
		byteValue := packet[currentPosition]
		if byteValue == 0 {
			name = append(name, 0)
			if !pointerFound {
				endOffset = currentPosition + 1
			}
			break
		}

		if byteValue&0xC0 == 0xC0 {
			if !pointerFound {
				endOffset = currentPosition + 2
				pointerFound = true
			}
			currentPosition = int(binary.BigEndian.Uint16(packet[currentPosition:currentPosition+2]) & 0x3FFF)
		} else {
			labelLength := int(byteValue)
			name = append(name, packet[currentPosition:currentPosition+labelLength+1]...)
			currentPosition += labelLength + 1
		}
	}
	return name, endOffset
}

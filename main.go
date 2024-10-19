package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

const (
	HOST = "127.0.0.1"
	PORT = 9001
	TYPE = "udp"
)

func handleIncomingRequests(conn *net.UDPConn, addr *net.UDPAddr) {
	println("Received a request: " + addr.String())
	headerBuffer := make([]byte, 1024)

	_, _, err := conn.ReadFromUDP(headerBuffer)
	if err != nil {
		log.Fatal(err)
	}

	var name string
	var reps uint32

	if headerBuffer[0] == byte(1) && headerBuffer[1023] == byte(0) {
		reps = binary.BigEndian.Uint32(headerBuffer[1:5])
		lengthOfName := binary.BigEndian.Uint32(headerBuffer[5:9])
		name = string(headerBuffer[9 : 9+lengthOfName])
	} else {
		log.Fatal("Invalid header")
	}

	conn.WriteToUDP([]byte("Header Received"), addr)

	dataBuffer := make([]byte, 1024)

	file, err := os.Create("./received/" + name)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < int(reps); i++ {
		_, _, err := conn.ReadFromUDP(dataBuffer)
		if err != nil {
			log.Fatal(err)
		}

		if dataBuffer[0] == byte(0) && dataBuffer[1023] == byte(1) {
			segmentNumber := dataBuffer[1:5]
			fmt.Printf("Segment Number: %d\n", binary.BigEndian.Uint32(segmentNumber))
			length := binary.BigEndian.Uint32(dataBuffer[5:9])
			fmt.Printf("File Data: %s\n", hex.EncodeToString(dataBuffer[9:9+length]))
			file.Write(dataBuffer[9 : 9+length])
		} else {
			log.Fatal("Invalid Segment")
		}

		conn.WriteToUDP([]byte("Segment Received"), addr)
	}

	time := time.Now().UTC().Format("Monday, 02-Jan-06 15:04:05 MST")
	conn.WriteToUDP([]byte(time), addr)

	file.Close()
}

func main() {
	addr := net.UDPAddr{
		Port: PORT,
		IP:   net.ParseIP(HOST),
	}
	listen, err := net.ListenUDP(TYPE, &addr)
	if err != nil {
		log.Fatal(err)
	}
	defer listen.Close()

	println("Server has started on PORT " + fmt.Sprint(PORT))

	for {
		buffer := make([]byte, 1024)
		_, clientAddr, err := listen.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal(err)
		}
		go handleIncomingRequests(listen, clientAddr)
	}
}

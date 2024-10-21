package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
)

const (
	HOST = "127.0.0.1"
	PORT = 9001
	TYPE = "udp"
)

type MetaData struct {
	Name     string
	FileSize int64
	Reps     int32
	Data     []byte
}

func (meta MetaData) metaDataToBytes() []byte {
	var metaBytes bytes.Buffer
	enc := gob.NewEncoder(&metaBytes)

	err := enc.Encode(meta)
	if err != nil {
		log.Fatal(err)
	}

	return metaBytes.Bytes()
}

func handleIncomingRequests(conn *net.UDPConn, addr *net.UDPAddr, buffer []byte) {
	println("Received a request: " + addr.String())

	//var name string
	//var reps uint32

	fmt.Printf("Received: %s\n", buffer)

	file, err := os.OpenFile(`C:\Users\gfanha\Downloads\Escala-Controle-Julho (11).xls`, os.O_RDONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		// Could not obtain stat, handle error
	}

	fmt.Printf("File Size: %d\n", fi.Size())

	//if buffer[0] == byte(1) && buffer[1023] == byte(0) {
	//	reps = binary.BigEndian.Uint32(buffer[1:5])
	//	lengthOfName := binary.BigEndian.Uint32(buffer[5:9])
	//	name = string(buffer[9 : 9+lengthOfName])
	//} else {
	//	log.Fatal("Invalid header")
	//}

	fileBuffer := make([]byte, fi.Size())

	_, errorBuffer := file.Read(fileBuffer)
	if errorBuffer != nil {
		return
	}

	headerSize := 124

	bodySize := int64(1024 - headerSize)

	fmt.Printf("Header Size: %d\n", headerSize)
	fmt.Printf("Body Size: %d\n", bodySize)

	quantidadeDeReps := fi.Size() / bodySize

	for i := int64(0); i <= quantidadeDeReps; i++ {

		var dataBuffer []byte
		fmt.Printf("Reps: %d\n", quantidadeDeReps-i)
		fmt.Printf("Body Size: %v", i*bodySize)
		if size := fi.Size() - i*bodySize; size < bodySize {
			dataBuffer = fileBuffer[i*bodySize : i*bodySize]
		} else {
			dataBuffer = fileBuffer[i*bodySize : i*bodySize+bodySize]
			fmt.Printf("Body Size: %v", len(dataBuffer))
		}

		responseBuffer := MetaData{
			Name:     "",
			FileSize: fi.Size(),
			Reps:     int32(quantidadeDeReps - i),
			Data:     dataBuffer,
		}.metaDataToBytes()
		conn.WriteToUDP(responseBuffer, addr)
	}

	//dataBuffer := MetaData{
	//	Name:     "calendar-clock.svg",
	//	FileSize: fi.Size(),
	//	Reps:     1,
	//	Data:     fileBuffer,
	//}.metaDataToBytes()
	//
	//conn.WriteToUDP(dataBuffer, addr)

	//file, err := os.Create("./received/" + name)
	//if err != nil {
	//	log.Fatal(err)
	//}

	//for i := 0; i < int(reps); i++ {
	//	_, _, err := conn.ReadFromUDP(dataBuffer)
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//
	//	if dataBuffer[0] == byte(0) && dataBuffer[1023] == byte(1) {
	//		segmentNumber := dataBuffer[1:5]
	//		fmt.Printf("Segment Number: %d\n", binary.BigEndian.Uint32(segmentNumber))
	//		length := binary.BigEndian.Uint32(dataBuffer[5:9])
	//		fmt.Printf("File Data: %s\n", hex.EncodeToString(dataBuffer[9:9+length]))
	//		file.Write(dataBuffer[9 : 9+length])
	//	} else {
	//		log.Fatal("Invalid Segment")
	//	}
	//
	//	conn.WriteToUDP([]byte("Segment Received"), addr)
	//}

	//time := time.Now().UTC().Format("Monday, 02-Jan-06 15:04:05 MST")
	//conn.WriteToUDP([]byte(time), addr)

	//file.Close()
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
		go handleIncomingRequests(listen, clientAddr, buffer)
	}
}

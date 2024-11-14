package main

import (
	"bytes"
	"encoding/gob"
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

type ResponseMetaData struct {
	Name     string
	FileSize int64
	Reps     uint32
	Data     []byte
}

type Packet struct {
	Reps uint32
	Data []byte
}

type RequestMetaData struct {
	Name string
	Reps uint32
	Miss bool
}

func (meta ResponseMetaData) ResponseMetaDataToBytes() []byte {
	var metaBytes bytes.Buffer
	enc := gob.NewEncoder(&metaBytes)

	err := enc.Encode(meta)
	if err != nil {
		log.Fatal(err)
	}

	return metaBytes.Bytes()
}

func (meta RequestMetaData) RequestMetaDataToBytes() []byte {
	var metaBytes bytes.Buffer
	enc := gob.NewEncoder(&metaBytes)

	err := enc.Encode(meta)
	if err != nil {
		log.Fatal(err)
	}
	return metaBytes.Bytes()
}

func (meta Packet) PacketToBytes() []byte {
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

	fmt.Printf("Received: %s\n", buffer)
	dec := gob.NewDecoder(bytes.NewReader(buffer))
	var request RequestMetaData
	// TODO - Check if the header is valid
	errorDecode := dec.Decode(&request)
	if errorDecode != nil {
		log.Fatal(errorDecode)
	}
	// TODO - Check if the file exist
	file, err := os.OpenFile(request.Name, os.O_RDONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("File Size: %d\n", fi.Size())
	fileBuffer := make([]byte, fi.Size())
	_, errorBuffer := file.Read(fileBuffer)
	if errorBuffer != nil {
		return
	}
	headerSize := 64
	bodySize := int64(1024 - headerSize)
	fmt.Printf("Header Size: %d\n", headerSize)
	fmt.Printf("Body Size: %d\n", bodySize)
	quantidadeDeReps := fi.Size() / bodySize

	if request.Miss {

	} else {
		if request.Reps == 0 {
			fileInformation := ResponseMetaData{
				Name:     fi.Name(),
				FileSize: fi.Size(),
				Reps:     uint32(quantidadeDeReps),
				Data:     []byte(""),
			}.ResponseMetaDataToBytes()
			_, err = conn.WriteToUDP(fileInformation, addr)
			if err != nil {
				log.Fatal(err)
			}

			time.Sleep(1 * time.Second)
		}

		quantidadeEnvio := func() int64 {
			if request.Reps == 0 {
				if fi.Size() < bodySize {
					return 0
				} else if fi.Size() < bodySize*10 {
					return quantidadeDeReps
				}
				return 9
			}
			if uint32(quantidadeDeReps)/(request.Reps) > 0 {
				return 9
			}
			return quantidadeDeReps % int64(request.Reps)
		}
		for i := int64(request.Reps); i <= quantidadeEnvio(); i++ {
			var dataBuffer []byte
			fmt.Printf("Reps: %d", quantidadeDeReps-i)
			fmt.Printf(" Size sended: %v", i*bodySize)
			if size := fi.Size() - i*bodySize; size < bodySize {
				dataBuffer = fileBuffer[i*bodySize : fi.Size()]
				fmt.Printf(" Body Size: %v\n", len(dataBuffer))
			} else {
				dataBuffer = fileBuffer[i*bodySize : i*bodySize+bodySize]
				fmt.Printf(" Body Size: %v\n", len(dataBuffer))
			}
			packetBuffer := Packet{
				Reps: uint32(quantidadeDeReps - i),
				Data: dataBuffer,
			}.PacketToBytes()
			conn.WriteToUDP(packetBuffer, addr)
		}
	}
}

func handleHelloRequest(conn *net.UDPConn, addr *net.UDPAddr, buffer []byte) {
	println("Received a request: " + addr.String())
	conn.WriteToUDP([]byte("Hello from server"), addr)

	time.Sleep(1 * time.Second)
	conn.WriteToUDP([]byte("Hello Again \n"), addr)
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

	for i := 0; ; i++ {
		fmt.Println("Valor de i: ", i)
		buffer := make([]byte, 1024)
		_, clientAddr, err := listen.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal(err)
		}
		//go handleHelloRequest(listen, clientAddr, buffer)
		go handleIncomingRequests(listen, clientAddr, buffer)
	}
}

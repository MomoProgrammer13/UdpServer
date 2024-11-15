package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"hash/crc32"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const (
	HOST       = "127.0.0.1"
	PORT       = 9001
	TYPE       = "udp"
	BUFFERSIZE = 2048
)

var FILES = map[string]string{
	"CR7":       "CR7.jpg",
	"DOC":       "Exemplo_Entrega_30_10.docx",
	"PDF":       "Matriz_981_2024.pdf",
	"ORDENACAO": "ordenacao.txt",
	"UTFPR":     "UTFPR campus Curitiba.mp4",
	"UFPR":      "Conheça a UFPR.mp4",
}

type ResponseMetaData struct {
	Name     string
	FileSize int64
	Reps     uint32
	Msg      string
}

type Packet struct {
	Reps     uint32
	Checksum uint32
	Data     []byte
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

func calculateChecksum(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

func handleIncomingRequests(conn *net.UDPConn, addr *net.UDPAddr, buffer []byte) {
	dec := gob.NewDecoder(bytes.NewReader(buffer))
	var request RequestMetaData
	// TODO - Check if the header is valid
	errorDecode := dec.Decode(&request)
	if errorDecode != nil {
		log.Fatal(errorDecode)
	}
	// TODO - Check if the file exist
	filename := FILES[strings.ToUpper(strings.ReplaceAll(request.Name, " ", ""))]
	if filename == "" {
		errorInformation := ResponseMetaData{
			Name: "__ERROR__",
			Msg:  "File not found",
		}.ResponseMetaDataToBytes()
		_, err := conn.WriteToUDP(errorInformation, addr)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	file, err := os.OpenFile(fmt.Sprintf("Files/%s", filename), os.O_RDONLY, 0755)
	if err != nil {
		errorInformation := ResponseMetaData{
			Name: "__ERROR__",
			Msg:  "File not found",
		}.ResponseMetaDataToBytes()
		_, err = conn.WriteToUDP(errorInformation, addr)
		return
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		errorInformation := ResponseMetaData{
			Name: "__ERROR__",
			Msg:  "Failed to get file information",
		}.ResponseMetaDataToBytes()
		_, err = conn.WriteToUDP(errorInformation, addr)
		return
	}

	headerSize := 128
	bodySize := int64(BUFFERSIZE - headerSize)
	quantidadeDeReps := fi.Size() / bodySize

	if request.Miss {
		missingPacket := request.Reps
		fmt.Println("Reenviando pacote", missingPacket)
		fileBuffer := make([]byte, bodySize)
		_, errorBuffer := file.ReadAt(fileBuffer, int64(missingPacket)*bodySize)
		if errorBuffer != nil {
			conn.WriteToUDP([]byte("Error"), addr)
			return
		}
		packetBuffer := Packet{
			Reps:     uint32(missingPacket),
			Checksum: calculateChecksum(fileBuffer),
			Data:     fileBuffer,
		}.PacketToBytes()
		conn.WriteToUDP(packetBuffer, addr)
	} else {
		if request.Reps == 0 {
			fmt.Printf("Sending %s to %s\n", filename, addr.String())
			fileInformation := ResponseMetaData{
				Name:     fi.Name(),
				FileSize: fi.Size(),
				Reps:     uint32(quantidadeDeReps),
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
			if uint32(quantidadeDeReps) > request.Reps+10 {
				return 9
			}
			return quantidadeDeReps % int64(request.Reps)
		}

		fileBytesToSend := func() (int64, int64) {
			if request.Reps == 0 {
				if fi.Size() < bodySize*10 {
					return fi.Size(), 0
				}
				return bodySize * 10, 0
			} else if size := fi.Size() - int64(request.Reps)*bodySize; size < bodySize {
				return bodySize * 10, bodySize * int64(request.Reps)
			}
			return fi.Size() - int64(request.Reps)*bodySize, bodySize * int64(request.Reps)
		}

		a, b := fileBytesToSend()

		fileBuffer := make([]byte, a)
		_, errorBuffer := file.ReadAt(fileBuffer, b)
		if errorBuffer != nil {
			return
		}
		tam := quantidadeEnvio()
		for i := int64(0); i <= tam; i++ {
			var dataBuffer []byte
			//if i == 0 {
			//	continue
			//}
			if size := a - i*bodySize; size < bodySize {
				fmt.Printf("Finish sending %s to %s\n", filename, addr.String())
				dataBuffer = fileBuffer[i*bodySize : a]
			} else {
				dataBuffer = fileBuffer[i*bodySize : i*bodySize+bodySize]
			}
			packetBuffer := Packet{
				Reps:     uint32(request.Reps + uint32(i)),
				Checksum: calculateChecksum(dataBuffer),
				Data:     dataBuffer,
			}.PacketToBytes()
			conn.WriteToUDP(packetBuffer, addr)
		}
	}
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
		buffer := make([]byte, BUFFERSIZE)
		_, clientAddr, err := listen.ReadFromUDP(buffer)
		if err != nil {
			log.Fatal(err)
		}
		//go handleHelloRequest(listen, clientAddr, buffer)
		go handleIncomingRequests(listen, clientAddr, buffer)
	}
}

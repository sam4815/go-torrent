package utils

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
	"net"
	"time"
)

type Peer struct {
	IP         net.IP
	Port       uint16
	Connection net.Conn
	Bitfield   Bitfield
}

func (peer *Peer) Handshake(torrent TorrentFile) error {
	log.Print("HANDSHAKIN' WITH ", fmt.Sprintf("%s:%d", peer.IP, peer.Port))
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", peer.IP, peer.Port), time.Second)

	if err != nil {
		return err
	}

	// Protocol identifier
	pstr := "BitTorrent protocol"
	// Generate peer ID
	peerID := sha1.Sum([]byte("-Tk8hj0wgej6ch"))

	handshakePacket := make([]byte, 68)
	copy(handshakePacket[0:1], []byte{uint8(len(pstr))}) // length of protocol identifier
	copy(handshakePacket[1:20], []byte(pstr))            // protocol identifier
	copy(handshakePacket[20:28], make([]byte, 8))        // extension support (all unsupported)
	copy(handshakePacket[28:48], torrent.InfoHash[:])    // info hash
	copy(handshakePacket[48:68], peerID[:])              // peer ID

	conn.Write(handshakePacket)
	resp := make([]byte, 2048)
	_, err = conn.Read(resp)

	if err != nil {
		return err
	}

	protocol := resp[0]
	infoHash := resp[28:48]

	if protocol != 19 || !bytes.Equal(torrent.InfoHash[:], infoHash) {
		return errors.New("invalid handshake response")
	}

	log.Print("GREAT HANDSHAKE SON")
	peer.Connection = conn

	return nil
}

func (peer Peer) SendMessage(message Message) (Message, error) {
	// log.Print(message.ID, len(message.Payload), len(message.ToBytes()), message.ToBytes())

	_, err := peer.Connection.Write(message.ToBytes())
	if err != nil {
		log.Print(err)
		return Message{}, err
	}
	peer.Connection.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

	return ToMessage(peer.Connection), nil
}

func (peer Peer) GetPiece(index int, length int) ([]byte, error) {
	blockDataChan := make(chan []byte)

	piece := make([]byte, 0)
	blockSize := 16384
	numBlocks := length/blockSize + 1
	log.Print(length, numBlocks, " BLOCKS")
	requestCount := make(map[int]int)
	allRequests := 0
	failed := 0

	blockIndexChan := make(chan int, numBlocks+1)

	for i := 0; i < 1; i++ {
		go func() {
			for {
				then := time.Now()
				blockIndex, more := <-blockIndexChan
				requestCount[blockIndex] += 1
				allRequests += 1

				if !more {
					return
				}

				requestMessage := RequestMessage(index, blockIndex, blockSize)
				// log.Print(requestMessage.ToBytes())
				responseMessage, err := peer.SendMessage(requestMessage)

				if err != nil || responseMessage.ID != MsgPiece {
					if err != nil {
						log.Print(err)
					}
					failed += 1
					blockIndexChan <- blockIndex
					time_elapsed := time.Since(then)
					log.Print("FAILURE TOOK ", time_elapsed, responseMessage.ID)
					continue
				}

				blockDataChan <- responseMessage.Payload[8:]
			}
		}()
	}

	for i := 0; i < numBlocks; i++ {
		blockIndexChan <- i
	}

	for i := 0; i < numBlocks; i++ {
		block := <-blockDataChan
		log.Print("RECEIVED BLOCK WITH LENGTH ", len(block))
	}
	close(blockIndexChan)
	log.Print(requestCount)
	log.Print(allRequests, failed)

	return piece, nil
}

func (peer Peer) Download(torrent TorrentFile) error {
	bitfield := CreateBitfield(len(torrent.PieceHash))
	bitfieldMessage := Message{
		ID:      MsgBitfield,
		Payload: bitfield,
	}

	peer.Connection.Write(bitfieldMessage.ToBytes())

	bitfieldResponse, err := peer.SendMessage(bitfieldMessage)
	if err != nil {
		return err
	}

	peer.Bitfield = bitfieldResponse.Payload

	piece, err := peer.GetPiece(0, torrent.PieceLength)
	if err != nil {
		return err
	}

	log.Print(piece)
	log.Print("PIECE LENGTH: ", len(piece))

	return nil
}

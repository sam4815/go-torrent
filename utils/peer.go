package utils

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

type Peer struct {
	IP         net.IP
	Port       uint16
	Connection net.Conn
	Bitfield   Bitfield
	Choked     bool
}

func (peer *Peer) Handshake(torrent TorrentFile) error {
	log.Print("HANDSHAKIN' WITH ", fmt.Sprintf("%s:%d", peer.IP, peer.Port))
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", peer.IP, peer.Port))

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
	resp := make([]byte, 68)
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
	peer.Choked = true

	return nil
}

func (peer Peer) SendMessage(message Message) error {
	log.Print(message.ID, len(message.Payload), len(message.ToBytes()), message.ToBytes())

	_, err := peer.Connection.Write(message.ToBytes())
	if err != nil {
		log.Print(err)
		return err
	}

	return nil
}

func (peer Peer) Flush() {
	buffer := make([]byte, 2048)
	peer.Connection.Read(buffer)
	log.Print("FLUSHED: ", buffer)
}

func (peer Peer) GetPiece(index int, torrent TorrentFile) ([]byte, error) {
	blockDataChan := make(chan []byte)

	pieceSize := torrent.PieceLength
	remainingBytes := torrent.Length - (index * torrent.PieceLength)
	if remainingBytes < pieceSize {
		pieceSize = remainingBytes
	}

	piece := make([]byte, pieceSize)
	blockSize := 16384
	numBlocks := 1 + (pieceSize-1)/blockSize

	blockIndexChan := make(chan int, numBlocks+1)

	for i := 0; i < 1; i++ {
		go func() {
			for {
				blockIndex, more := <-blockIndexChan

				if !more {
					return
				}

				remainingBytes := torrent.Length - ((index * torrent.PieceLength) + (blockIndex * blockSize))
				if remainingBytes < blockSize {
					blockSize = remainingBytes
				}

				peer.SendMessage(RequestMessage(index, blockIndex*blockSize, blockSize))
				responseMessage, _ := ReadMessage(peer.Connection)
				for responseMessage.ID != MsgPiece {
					time.Sleep(time.Second)
					responseMessage, _ = ReadMessage(peer.Connection)
				}

				blockDataChan <- responseMessage.Payload
			}
		}()
	}

	for i := 0; i < numBlocks; i++ {
		blockIndexChan <- i
	}

	for i := 0; i < numBlocks; i++ {
		block := <-blockDataChan
		offset := binary.BigEndian.Uint32(block[4:8])
		copy(piece[offset:], block[8:])
	}

	close(blockIndexChan)

	return piece, nil
}

func (peer Peer) Download(torrent TorrentFile) error {
	bitfield := CreateBitfield(len(torrent.PieceHash))
	bitfieldMessage := Message{
		ID:      MsgBitfield,
		Payload: bitfield,
	}

	peer.SendMessage(bitfieldMessage)
	bitfieldResponse, err := ReadMessage(peer.Connection)
	if err != nil {
		return err
	}
	log.Print("BITFIELD RESPONSE: ", bitfieldResponse)

	peer.Bitfield = bitfieldResponse.Payload

	peer.SendMessage(InterestedMessage())

	for peer.Choked {
		message, err := ReadMessage(peer.Connection)
		if err != nil {
			return err
		}
		if message.ID == 1 {
			peer.Choked = false
		}
	}

	download := make([]byte, torrent.Length)

	for index, hash := range torrent.PieceHash {
		log.Print("Attempting piece ", index)
		piece, err := peer.GetPiece(index, torrent)
		if err != nil {
			return err
		}

		pieceHash := sha1.Sum(piece)
		if hash != pieceHash {
			log.Fatal("Invalid piece")
		}

		copy(download[(index*torrent.PieceLength):], piece)
	}

	err = os.WriteFile("final.pdf", download, 0644)
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

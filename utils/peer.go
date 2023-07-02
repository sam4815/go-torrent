package utils

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
)

type Peer struct {
	IP         net.IP
	Port       uint16
	Connection net.Conn
	Bitfield   Bitfield
	Choked     bool
}

func (peer *Peer) Handshake(torrent TorrentFile) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", peer.IP, peer.Port))

	if err != nil {
		return err
	}

	pstr := "BitTorrent protocol"
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

	peer.Connection = conn
	peer.Choked = true

	return nil
}

func (peer Peer) SendMessage(message Message) error {
	// log.Print(message.ID, len(message.Payload), len(message.ToBytes()), message.ToBytes())

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
			}
		}()
	}

	for i := 0; i < numBlocks; i++ {
		blockIndexChan <- i
	}

	for i := 0; i < numBlocks; i++ {
		block, _ := ReadMessage(peer.Connection)
		offset := binary.BigEndian.Uint32(block.Payload[4:8])
		copy(piece[offset:], block.Payload[8:])
	}

	close(blockIndexChan)

	return piece, nil
}

func (peer *Peer) AnnounceInterested(torrent TorrentFile) error {
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

	peer.Bitfield = bitfieldResponse.Payload

	peer.SendMessage(InterestedMessage())

	for peer.Choked {
		message, err := ReadMessage(peer.Connection)
		if err != nil {
			return err
		}

		peer.Choked = message.ID != 1
	}

	return nil
}

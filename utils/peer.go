package utils

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
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
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", peer.IP, peer.Port), time.Second*3)

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

	return nil
}

func (peer Peer) SendMessage(message Message) error {
	Debugf("Sending message with length %d and ID %d to peer with IP %s", len(message.Payload), message.ID, peer.IP)

	_, err := peer.Connection.Write(message.ToBytes())
	if err != nil {
		return err
	}

	return nil
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

				offset := blockIndex * blockSize
				remainingBlockBytes := remainingBytes - offset
				if remainingBlockBytes < blockSize {
					blockSize = remainingBlockBytes
				}

				peer.SendMessage(RequestMessage(index, offset, blockSize))
			}
		}()
	}

	for i := 0; i < numBlocks; i++ {
		blockIndexChan <- i
	}

	for i := 0; i < numBlocks; i++ {
		message, err := ReadMessage(peer.Connection)
		if err != nil {
			return nil, err
		}

		if message.ID != MsgPiece {
			return nil, errors.New("failed to receive piece message")
		}

		offset := binary.BigEndian.Uint32(message.Payload[4:8])
		copy(piece[offset:], message.Payload[8:])
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

	return nil
}

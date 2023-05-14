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
	log.Print(message.ID, len(message.Payload), len(message.ToBytes()), message.ToBytes())
	resp := make([]byte, 2048)

	peer.Connection.Write(message.ToBytes())

	peer.Connection.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, err := peer.Connection.Read(resp)

	if err != nil {
		log.Print("ERR: ", err)
		return Message{}, err
	}

	return ToMessage(bytes.NewReader(resp)), nil
}

func (peer Peer) GetPiece(index int, length int) ([]byte, error) {
	piece := make([]byte, 0)
	offset := 1
	block_size := 16384
	num_blocks := 1 + length/block_size

	log.Print("REQ ", num_blocks, " BLOCKS")

	for offset <= num_blocks {
		log.Print("w")
		requestMessage := RequestMessage(index, offset, block_size)
		requestResponse, err := peer.SendMessage(requestMessage)
		if err != nil {
			return piece, err
		}
		if requestResponse.ID != MsgPiece {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		log.Print("RES ID: ", requestResponse.ID, "RES LEN: ", len(requestResponse.Payload))
		piece = append(piece, requestResponse.Payload...)
		time.Sleep(500 * time.Millisecond)
		offset += 1
	}

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

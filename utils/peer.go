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

func (peer Peer) Download(torrent TorrentFile) error {
	bitfield := CreateBitfield(torrent.Length)
	bitfieldMessage := Message{
		ID:      MsgBitfield,
		Payload: bitfield,
	}

	peer.Connection.Write(bitfieldMessage.Serialize())

	resp := make([]byte, 2048)
	bitfieldResponse := ToMessage(bytes.NewReader(resp))
	log.Print("Length of torrent: ", torrent.Length)
	for bitfieldResponse.ID == MsgChoke {
		_, err := peer.Connection.Read(resp)
		if err != nil {
			return err
		}

		bitfieldResponse = ToMessage(bytes.NewReader(resp))
		log.Print("RESPONSE:", len(bitfieldResponse.Payload))
		time.Sleep(time.Second * 2)
	}

	log.Print("RESPONSE:", bitfieldResponse.ID)

	return nil
}

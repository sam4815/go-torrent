package utils

import (
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
)

type Peer struct {
	IP   net.IP
	Port uint16
}

type Tracker struct {
	Host         string
	Port         string
	ConnectionID uint64
}

type Download struct {
	Peers      []Peer
	Downloaded uint64
	Uploaded   uint64
	Leechers   uint32
	Seeders    uint32
	Interval   uint32
}

func ParsePeers(peersBytes []byte) []Peer {
	peers := make([]Peer, 0)

	for offset := 0; offset < len(peersBytes); offset += 6 {
		peer := Peer{
			IP:   net.IP(peersBytes[offset : offset+4]),
			Port: binary.BigEndian.Uint16(peersBytes[offset+4 : offset+6]),
		}

		peers = append(peers, peer)
	}

	return peers
}

func (tracker *Tracker) Connect() error {
	conn, _ := net.Dial("udp", fmt.Sprintf("%s:%s", tracker.Host, tracker.Port))
	defer conn.Close()

	// Generate random transaction ID
	transactionID := rand.Uint32()

	// Connection request packet
	connectPacket := make([]byte, 16)
	binary.BigEndian.PutUint64(connectPacket[0:8], 0x41727101980)
	binary.BigEndian.PutUint32(connectPacket[8:12], 0) // action = 0 (connect)
	binary.BigEndian.PutUint32(connectPacket[12:16], transactionID)

	// Send the UDP request
	conn.Write(connectPacket)
	// Read the response
	resp := make([]byte, 16)
	_, err := conn.Read(resp)

	if err != nil {
		return err
	}

	// Parse response
	action := binary.BigEndian.Uint32(resp[0:4])
	respTransactionID := binary.BigEndian.Uint32(resp[4:8])
	connectionID := binary.BigEndian.Uint64(resp[8:16])

	if action != 0 || transactionID != respTransactionID {
		return errors.New("invalid connect response")
	}

	tracker.ConnectionID = connectionID

	return nil
}

func (tracker Tracker) Announce(torrent TorrentFile) (Download, error) {
	conn, _ := net.Dial("udp", fmt.Sprintf("%s:%s", tracker.Host, tracker.Port))
	defer conn.Close()

	// Generate random transaction ID
	transactionID := rand.Uint32()
	// Generate peer ID
	peerID := sha1.Sum([]byte("-TR2940-k8hj0wgej6ch"))

	announcePacket := make([]byte, 98)
	binary.BigEndian.PutUint64(announcePacket[0:8], tracker.ConnectionID)     // connection ID
	binary.BigEndian.PutUint32(announcePacket[8:12], 1)                       // action = 1 (announce)
	binary.BigEndian.PutUint32(announcePacket[12:16], transactionID)          // transaction ID
	copy(announcePacket[16:36], torrent.InfoHash[:])                          // info hash
	copy(announcePacket[36:56], peerID[:])                                    // peer ID
	binary.BigEndian.PutUint64(announcePacket[56:64], 0)                      // downloaded
	binary.BigEndian.PutUint64(announcePacket[64:72], uint64(torrent.Length)) // left
	binary.BigEndian.PutUint64(announcePacket[72:80], 0)                      // uploaded
	binary.BigEndian.PutUint32(announcePacket[80:84], 0)                      // event = 0 (none)
	binary.BigEndian.PutUint32(announcePacket[84:88], 0)                      // ip address
	binary.BigEndian.PutUint32(announcePacket[88:92], 53)                     // key
	binary.BigEndian.PutUint32(announcePacket[92:96], ^uint32(0))             // desired number of peers = -1 (default)
	binary.BigEndian.PutUint16(announcePacket[96:98], 1337)                   // port

	conn.Write(announcePacket)
	resp := make([]byte, 2048)
	numBytes, _ := conn.Read(resp)

	// Parse response
	action := binary.BigEndian.Uint32(resp[0:4])
	respTransactionID := binary.BigEndian.Uint32(resp[4:8])

	if action != 1 || transactionID != respTransactionID {
		return Download{}, errors.New("invalid connect response")
	}

	download := Download{
		Interval: binary.BigEndian.Uint32(resp[8:12]),
		Leechers: binary.BigEndian.Uint32(resp[12:16]),
		Seeders:  binary.BigEndian.Uint32(resp[16:20]),
		Peers:    ParsePeers(resp[20:numBytes]),
	}

	return download, nil
}

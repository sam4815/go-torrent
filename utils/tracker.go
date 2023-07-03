package utils

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Tracker struct {
	AnnounceURL *url.URL
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

func (tracker *Tracker) AnnounceUDP(torrent TorrentFile) ([]Peer, error) {
	conn, err := net.DialTimeout("udp", tracker.AnnounceURL.Host, time.Second*2)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	transactionID := rand.Uint32()
	connectPacket := make([]byte, 16)
	binary.BigEndian.PutUint64(connectPacket[0:8], 0x41727101980)   // Bittorent protocol ID
	binary.BigEndian.PutUint32(connectPacket[8:12], 0)              // action = 0 (connect)
	binary.BigEndian.PutUint32(connectPacket[12:16], transactionID) // random transaction ID

	conn.Write(connectPacket)
	resp := make([]byte, 2048)
	_, err = conn.Read(resp)

	if err != nil {
		return nil, err
	}

	action := binary.BigEndian.Uint32(resp[0:4])
	respTransactionID := binary.BigEndian.Uint32(resp[4:8])
	connectionID := binary.BigEndian.Uint64(resp[8:16])

	if action != 0 || transactionID != respTransactionID {
		return nil, errors.New("invalid UDP connect response")
	}

	announceMessage := GenerateAnnounceMessage(torrent)
	announceMessage.ConnectionID = connectionID
	announceMessage.TransactionID = transactionID

	announcePacket := announceMessage.ToBytes()
	conn.Write(announcePacket)
	resp = make([]byte, 2048)
	numBytes, _ := conn.Read(resp)

	action = binary.BigEndian.Uint32(resp[0:4])
	respTransactionID = binary.BigEndian.Uint32(resp[4:8])

	if action != 1 || transactionID != respTransactionID {
		return nil, errors.New("invalid UDP announce response")
	}

	return ParsePeers(resp[20:numBytes]), nil
}

func (tracker Tracker) AnnounceTCP(torrent TorrentFile) ([]Peer, error) {
	tracker.AnnounceURL.RawQuery = GenerateAnnounceMessage(torrent).ToQueryParams()

	httpResp, err := http.Get(tracker.AnnounceURL.String())
	if err != nil {
		return nil, err
	}

	defer httpResp.Body.Close()

	announceResponse, err := Announce(httpResp.Body)
	if err != nil {
		return nil, err
	}

	peers := make([]Peer, 0)
	for _, peer := range announceResponse.Peers {
		peers = append(peers, Peer{IP: net.ParseIP(peer.IP), Port: uint16(peer.Port)})
	}

	return peers, nil
}

func (tracker Tracker) Announce(torrent TorrentFile) ([]Peer, error) {
	if tracker.AnnounceURL.Scheme == "udp" {
		return tracker.AnnounceUDP(torrent)
	}

	if tracker.AnnounceURL.Scheme == "https" || tracker.AnnounceURL.Scheme == "http" {
		return tracker.AnnounceTCP(torrent)
	}

	return nil, errors.New("unsupported url scheme")
}

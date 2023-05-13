package utils

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"net/url"
)

type AnnounceMessage struct {
	ConnectionID  uint64
	Action        uint32
	TransactionID uint32
	InfoHash      [20]byte
	PeerID        [20]byte
	Downloaded    uint64
	Left          uint64
	Uploaded      uint64
	Event         uint32
	IP            uint32
	Key           uint32
	NumWant       uint32
	Port          uint16
}

func GenerateAnnounceMessage(torrent TorrentFile) AnnounceMessage {
	return AnnounceMessage{
		Action:   1,
		InfoHash: torrent.InfoHash,
		PeerID:   sha1.Sum([]byte("-TR2940-k8hj0wgej6ch")),
		Left:     uint64(torrent.Length),
		NumWant:  ^uint32(0),
		Port:     1337,
	}
}

func (m AnnounceMessage) ToBytes() []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, m)

	return buf.Bytes()
}

func (m AnnounceMessage) ToQueryParams() string {
	q := url.Values{}
	q.Add("info_hash", string(m.InfoHash[:]))
	q.Add("peer_id", string(m.PeerID[:]))
	q.Add("port", fmt.Sprint(m.Port))
	q.Add("uploaded", fmt.Sprint(m.Uploaded))
	q.Add("downloaded", fmt.Sprint(m.Downloaded))
	q.Add("left", fmt.Sprint(m.Left))

	return q.Encode()
}

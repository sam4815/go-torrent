package utils

import (
	"bytes"
	"crypto/sha1"
	"io"
	"log"

	"github.com/jackpal/bencode-go"
)

type BencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type BencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     BencodeInfo `bencode:"info"`
}

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHash   [][20]byte
	PieceLength int
	Length      int
	Name        string
}

func Open(r io.Reader) (*BencodeTorrent, error) {
	bto := BencodeTorrent{}
	err := bencode.Unmarshal(r, &bto)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &bto, nil
}

func (b BencodeTorrent) ToTorrentFile() TorrentFile {
	hashes := [][20]byte{}
	for i := 0; i < len(b.Info.Pieces); i += 20 {
		var hash [20]byte
		copy(hash[:], []byte(b.Info.Pieces[i:i+20]))
		hashes = append(hashes, hash)
	}

	var infoBuffer bytes.Buffer
	bencode.Marshal(&infoBuffer, b.Info)

	return TorrentFile{
		Announce:    b.Announce,
		PieceLength: b.Info.PieceLength,
		Length:      b.Info.Length,
		Name:        b.Info.Name,
		InfoHash:    sha1.Sum(infoBuffer.Bytes()),
		PieceHash:   hashes,
	}
}

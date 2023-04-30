package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

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

func udpRequest(url string) (string, error) {
	// Resolve the address of the server
	addr, err := net.ResolveUDPAddr("udp", "tracker.ccc.de:80")
	if err != nil {
		log.Print("HERE")
		return "", err
	}
	log.Print(addr)

	// Open a UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Send the UDP request
	_, err = conn.Write([]byte("Hello, server!"))
	if err != nil {
		return "", err
	}

	// Read the response
	buffer := make([]byte, 1024)
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		return "", err
	}

	// Print the response
	return string(buffer[:n]), nil
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

func (b BencodeTorrent) ToTorrentFile() (TorrentFile, error) {
	hashes := [][20]byte{}
	for i := 0; i < len(b.Info.Pieces); i += 20 {
		var hash [20]byte
		copy(hash[:], []byte(b.Info.Pieces[i:i+20]))
		hashes = append(hashes, hash)
	}

	var infoBuffer bytes.Buffer
	bencode.Marshal(&infoBuffer, b.Info)

	// log.Print(hashes[2])

	return TorrentFile{
		Announce:    b.Announce,
		PieceLength: b.Info.PieceLength,
		Length:      b.Info.Length,
		Name:        b.Info.Name,
		InfoHash:    sha1.Sum(infoBuffer.Bytes()),
		PieceHash:   hashes,
	}, nil
}

func (t *TorrentFile) BuildTrackerURL(peerID [20]byte, port uint16) string {
	base, _ := url.Parse(t.Announce)
	params := url.Values{
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(int(t.Length))},
	}

	base.RawQuery = params.Encode()
	return base.String()
}

func main() {
	r, err := os.Open("mac.torrent")
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	reader := bufio.NewReader(r)

	benconded_torrent, _ := Open(reader)
	torrent, _ := benconded_torrent.ToTorrentFile()
	trackerURL := torrent.BuildTrackerURL(sha1.Sum([]byte("fuck")), 1337)

	log.Print(strings.Split(trackerURL, "://")[1])
	res, err := udpRequest(strings.Split(trackerURL, "://")[1])
	if err != nil {
		log.Fatal(err)
	}

	log.Print(res)
}

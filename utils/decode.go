package utils

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"io"
	"os"

	"github.com/jackpal/bencode-go"
)

type BencodeFile struct {
	Length int      `bencode:"length"`
	Path   []string `beconde:"path"`
}

type BencodeInfo struct {
	Files       []BencodeFile `bencode:"files"`
	Pieces      string        `bencode:"pieces"`
	PieceLength int           `bencode:"piece length"`
	Length      int           `bencode:"length"`
	Name        string        `bencode:"name"`
}

type BencodeTorrent struct {
	Announce     string      `bencode:"announce"`
	AnnounceList [][]string  `bencode:"announce-list"`
	Info         BencodeInfo `bencode:"info"`
}

type BencodePeer struct {
	IP   string `bencode:"ip"`
	Port int    `bencode:"port"`
}

type BencodeAnnounce struct {
	Interval int           `bencode:"interval"`
	Peers    []BencodePeer `bencode:"peers"`
}

type File struct {
	Length int
	Path   []string
}

type TorrentFile struct {
	AnnounceList []string
	InfoHash     [20]byte
	PieceHash    [][20]byte
	PieceLength  int
	Length       int
	Name         string
	Files        []File
}

func DecodeBencodedFile(file *os.File) (TorrentFile, error) {
	reader := bufio.NewReader(file)

	var infoBuffer bytes.Buffer
	fileReader := io.TeeReader(reader, &infoBuffer)

	bto := BencodeTorrent{}
	err := bencode.Unmarshal(fileReader, &bto)

	if err != nil {
		return TorrentFile{}, err
	}

	torrent := bto.ToTorrentFile()

	decoded, _ := bencode.Decode(bufio.NewReader(&infoBuffer))

	if decodedMap, ok := decoded.(map[string]any); ok {
		var infoBuffer bytes.Buffer
		bencode.Marshal(&infoBuffer, decodedMap["info"])

		torrent.InfoHash = sha1.Sum(infoBuffer.Bytes())
	}

	return torrent, nil
}

func Announce(r io.Reader) (*BencodeAnnounce, error) {
	ba := BencodeAnnounce{}
	err := bencode.Unmarshal(r, &ba)

	if err != nil {
		return nil, err
	}

	return &ba, nil
}

func (b BencodeTorrent) ToTorrentFile() TorrentFile {
	hashes := [][20]byte{}
	for i := 0; i < len(b.Info.Pieces); i += 20 {
		var hash [20]byte
		copy(hash[:], []byte(b.Info.Pieces[i:i+20]))
		hashes = append(hashes, hash)
	}

	announceList := []string{b.Announce}
	for _, announceUrl := range b.AnnounceList {
		announceList = append(announceList, announceUrl...)
	}

	length := b.Info.Length
	files := make([]File, 0)
	for _, file := range b.Info.Files {
		files = append(files, File(file))
		length += file.Length
	}

	return TorrentFile{
		Name:         b.Info.Name,
		PieceLength:  b.Info.PieceLength,
		AnnounceList: announceList,
		Length:       length,
		PieceHash:    hashes,
		Files:        files,
	}
}

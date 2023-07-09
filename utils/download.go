package utils

import (
	"crypto/sha1"
	"os"
	"strings"
)

type Download struct {
	ConnectedCountries []string
	Torrent            TorrentFile
	CompletedPieceHash [][20]byte
	PieceIndexChan     chan int
	Completed          chan bool
}

func StartDownload(torrent TorrentFile) (*Download, error) {
	download := &Download{
		Torrent:            torrent,
		PieceIndexChan:     make(chan int, 50),
		ConnectedCountries: make([]string, 0),
		Completed:          make(chan bool, 1),
	}

	go func() {
		for i := 0; i < len(torrent.PieceHash); i++ {
			download.PieceIndexChan <- i
		}
	}()

	return download, nil
}

func (download *Download) AddPeer(peer Peer) {
	err := peer.Handshake(download.Torrent)
	if err != nil {
		Debugf("Handshake failed: %s", peer.IP.String())
		return
	}

	err = peer.AnnounceInterested(download.Torrent)
	if err != nil {
		Debugf("Failed to initiate download: %s", peer.IP.String())
		return
	}

	download.ConnectedCountries = append(download.ConnectedCountries, GetCountryCode(peer))

	for {
		pieceIndex, more := <-download.PieceIndexChan
		if !more {
			return
		}

		if !peer.Bitfield.HasPiece(pieceIndex) {
			download.PieceIndexChan <- pieceIndex
			continue
		}

		Debugf("Requesting piece with index %d from peer with IP %s", pieceIndex, peer.IP.String())

		piece, err := peer.GetPiece(pieceIndex, download.Torrent)
		if err != nil {
			Debugf("Error requesting piece: %s", err)

			download.PieceIndexChan <- pieceIndex
			continue
		}

		pieceHash := sha1.Sum(piece)
		if download.Torrent.PieceHash[pieceIndex] != pieceHash {
			Debugf("Invalid piece from IP %s with index %d", peer.IP.String(), pieceIndex)

			download.PieceIndexChan <- pieceIndex
			continue
		}

		err = download.WriteAt(pieceIndex*download.Torrent.PieceLength, piece)
		if err != nil {
			Debugf("Error writing piece from IP %s with index %d: %s", peer.IP.String(), pieceIndex, err)

			download.PieceIndexChan <- pieceIndex
			continue
		}

		download.CompletedPieceHash = append(download.CompletedPieceHash, pieceHash)

		if len(download.CompletedPieceHash) == len(download.Torrent.PieceHash) {
			download.Completed <- true
		}
	}
}

func (download Download) Close() {
	close(download.PieceIndexChan)
}

func WriteAtFile(path []string, offset int, fileBytes []byte) error {
	fileDir := strings.Join(path[:len(path)-1], "/")
	filePath := strings.Join(path, "/")

	if len(fileDir) > 0 {
		err := os.MkdirAll(fileDir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteAt(fileBytes, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func (download Download) WriteAt(offset int, piece []byte) error {
	if len(download.Torrent.Files) == 0 {
		err := WriteAtFile([]string{download.Torrent.Name}, offset, piece)
		return err
	}

	fileOffset := 0

	for _, file := range download.Torrent.Files {
		filePath := append([]string{download.Torrent.Name}, file.Path...)
		fileMin := fileOffset
		fileMax := fileMin + file.Length

		pieceBeginsInFile := fileMin <= offset && offset < fileMax
		pieceEndsInFile := fileMin <= (offset+len(piece)) && (offset+len(piece)) < fileMax

		if pieceBeginsInFile && pieceEndsInFile {
			err := WriteAtFile(filePath, offset-fileMin, piece)
			return err
		}

		if pieceBeginsInFile {
			pieceLength := fileMax - offset
			err := WriteAtFile(filePath, offset-fileMin, piece[:pieceLength])
			if err != nil {
				return err
			}

			return download.WriteAt(offset+pieceLength, piece[pieceLength:])
		}

		fileOffset += file.Length
	}

	return nil
}

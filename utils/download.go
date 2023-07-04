package utils

import (
	"crypto/sha1"
	"fmt"
	"os"
	"strings"
)

type Download struct {
	Peers              []Peer
	NumConnectedPeers  int
	Torrent            TorrentFile
	CompletedPieceHash [][20]byte
	PieceIndexChan     chan int
}

func StartDownload(peers []Peer, torrent TorrentFile) (*Download, error) {
	download := &Download{
		Peers:          peers,
		Torrent:        torrent,
		PieceIndexChan: make(chan int, 50),
	}

	for _, peer := range download.Peers {
		go func(peer Peer, download *Download) {
			err := peer.Handshake(download.Torrent)
			if err != nil {
				// log.Print("Handshake failed: ", peer.IP.String())
				return
			}

			err = peer.AnnounceInterested(download.Torrent)
			if err != nil {
				// log.Print("Failed to initiate download: ", peer.IP.String())
				return
			}

			download.NumConnectedPeers += 1

			for {
				pieceIndex, more := <-download.PieceIndexChan
				if !more {
					return
				}

				if !peer.Bitfield.HasPiece(pieceIndex) {
					download.PieceIndexChan <- pieceIndex
					continue
				}

				// log.Printf("Requesting piece with index %d from peer with IP %s", pieceIndex, peer.IP.String())
				piece, err := peer.GetPiece(pieceIndex, download.Torrent)
				if err != nil {
					// log.Print("Error requesting piece: ", err)
					download.PieceIndexChan <- pieceIndex
					continue
				}

				pieceHash := sha1.Sum(piece)
				if download.Torrent.PieceHash[pieceIndex] != pieceHash {
					// log.Print("Invalid piece: ", peer.IP.String(), pieceIndex)
					download.PieceIndexChan <- pieceIndex
					continue
				}

				err = download.WriteAt(pieceIndex*download.Torrent.PieceLength, piece)
				if err != nil {
					// log.Print("Error writing piece: ", err, peer.IP.String(), pieceIndex)
					download.PieceIndexChan <- pieceIndex
					continue
				}

				download.CompletedPieceHash = append(download.CompletedPieceHash, pieceHash)
			}
		}(peer, download)
	}

	go func() {
		for i := 0; i < len(torrent.PieceHash); i++ {
			download.PieceIndexChan <- i
		}
	}()

	return download, nil
}

func (download Download) Completed() bool {
	return len(download.CompletedPieceHash) == len(download.Torrent.PieceHash)
}

func (download Download) Progress() string {
	percentComplete := float64(len(download.CompletedPieceHash)) / float64(len(download.Torrent.PieceHash))

	progressBarSize := 40
	numCompletedBlocks := int(percentComplete * float64(progressBarSize))
	numEmptyBlocks := progressBarSize - numCompletedBlocks

	completedBlocks := strings.Repeat("█", numCompletedBlocks)
	emptyBlocks := strings.Repeat(" ", numEmptyBlocks)

	return fmt.Sprintf("[%s%s] %.2f %% // Connected to %d peers", completedBlocks, emptyBlocks, percentComplete*100, download.NumConnectedPeers)
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

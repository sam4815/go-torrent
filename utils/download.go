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
	Blob               []byte
	CompletedPieceHash [][20]byte
	PieceIndexChan     chan int
}

func StartDownload(peers []Peer, torrent TorrentFile) (*Download, error) {
	download := &Download{
		Peers:          peers,
		Torrent:        torrent,
		Blob:           make([]byte, torrent.Length),
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

				copy(download.Blob[(pieceIndex*download.Torrent.PieceLength):], piece)
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

func (download Download) WriteFiles() error {
	if len(download.Torrent.Files) == 0 {
		err := os.WriteFile(download.Torrent.Name, download.Blob, 0644)

		if err != nil {
			return err
		}
	} else {
		blobOffset := 0

		for _, file := range download.Torrent.Files {
			path := strings.Join(append([]string{download.Torrent.Name}, file.Path...), "/")
			folderPath := strings.Join(append([]string{download.Torrent.Name}, file.Path[:len(file.Path)-1]...), "/")

			blobSlice := download.Blob[blobOffset:(blobOffset + file.Length)]
			blobOffset += file.Length

			os.MkdirAll(folderPath, os.ModePerm)
			err := os.WriteFile(path, blobSlice, 0644)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

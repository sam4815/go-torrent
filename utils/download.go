package utils

import (
	"crypto/sha1"
	"log"
	"time"
)

type Download struct {
	Peers              []Peer
	Torrent            TorrentFile
	File               []byte
	CompletedPieceHash [][20]byte
}

func (download *Download) Start() error {
	download.File = make([]byte, download.Torrent.Length)
	pieceIndexChan := make(chan int, download.Torrent.PieceLength)

	for _, peer := range download.Peers {
		go func(peer Peer) {
			err := peer.Handshake(download.Torrent)
			if err != nil {
				log.Print("Handshake failed: ", peer.IP.String())
				return
			}

			err = peer.AnnounceInterested(download.Torrent)
			if err != nil {
				log.Print("Failed to initiate download: ", peer.IP.String())
				return
			}

			for {
				pieceIndex, more := <-pieceIndexChan
				if !more {
					return
				}

				if !peer.Bitfield.HasPiece(pieceIndex) {
					pieceIndexChan <- pieceIndex
					continue
				}

				log.Printf("Requesting piece with index %d from peer with IP %s", pieceIndex, peer.IP.String())
				piece, err := peer.GetPiece(pieceIndex, download.Torrent)
				if err != nil {
					pieceIndexChan <- pieceIndex
					continue
				}

				pieceHash := sha1.Sum(piece)
				if download.Torrent.PieceHash[pieceIndex] != pieceHash {
					log.Print("Invalid piece: ", peer.IP.String())
					pieceIndexChan <- pieceIndex
					continue
				}

				copy(download.File[(pieceIndex*download.Torrent.PieceLength):], piece)
				download.CompletedPieceHash = append(download.CompletedPieceHash, pieceHash)
			}
		}(peer)
	}

	for i := 0; i < len(download.Torrent.PieceHash); i++ {
		pieceIndexChan <- i
	}

	for len(download.CompletedPieceHash) != len(download.Torrent.PieceHash) {
		time.Sleep(time.Millisecond * 500)
	}

	close(pieceIndexChan)

	return nil
}

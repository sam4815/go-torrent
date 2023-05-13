package main

import (
	"bufio"
	"log"
	"net/url"
	"os"

	"go-torrent/utils"
)

func main() {
	r, _ := os.Open("bayes.torrent")
	defer r.Close()

	reader := bufio.NewReader(r)
	benconded_torrent, err := utils.Open(reader)
	if err != nil {
		log.Fatal(err)
	}

	torrent := benconded_torrent.ToTorrentFile()
	announceURL, _ := url.Parse(torrent.Announce)

	t := utils.Tracker{AnnounceURL: announceURL}

	peers, err := t.Announce(torrent)
	if err != nil {
		log.Fatal(err)
	}

	for _, peer := range peers {
		err = peer.Handshake(torrent)
		if err != nil {
			log.Print(err)
			continue
		}

		err = peer.Download(torrent)
		if err != nil {
			log.Print(err)
			continue
		}
	}
}

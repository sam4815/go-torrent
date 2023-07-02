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

	tracker := utils.Tracker{AnnounceURL: announceURL}

	peers, err := tracker.Announce(torrent)
	if err != nil {
		log.Fatal(err)
	}

	download := utils.Download{
		Peers:   peers,
		Torrent: torrent,
	}

	err = download.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(torrent.Name, download.File, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

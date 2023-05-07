package main

import (
	"bufio"
	"log"
	"net/url"
	"os"

	"go-torrent/utils"
)

func main() {
	r, _ := os.Open("nitrux.torrent")
	defer r.Close()

	reader := bufio.NewReader(r)
	benconded_torrent, err := utils.Open(reader)
	if err != nil {
		log.Fatal(err)
	}

	torrent := benconded_torrent.ToTorrentFile()
	announceURL, _ := url.Parse(torrent.Announce)

	t := utils.Tracker{
		Host: announceURL.Hostname(),
		Port: announceURL.Port(),
	}

	err = t.Connect()
	if err != nil {
		log.Fatal(err)
	}

	download, err := t.Announce(torrent)
	if err != nil {
		log.Fatal(err)
	}

	log.Print(download)
}

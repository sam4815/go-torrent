package main

import (
	"flag"
	"log"
	"os"
	"time"

	"go-torrent/utils"
)

func main() {
	filePath := flag.String("file", "", "torrent file path")
	flag.Parse()

	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatal("Error opening file: ", err)
	}
	defer file.Close()

	torrent, err := utils.DecodeBencodedFile(file)
	if err != nil {
		log.Fatal("Error decoding file: ", err)
	}

	trackers := utils.NewTrackers(torrent.AnnounceList)

	peers, err := trackers.Announce(torrent)
	if err != nil {
		log.Fatal("Error retrieving peers: ", err)
	}

	download, err := utils.StartDownload(peers, torrent)
	if err != nil {
		log.Fatal("Error initiating download: ", err)
	}
	defer download.Close()

	display := utils.StartDisplay(download, time.Millisecond*100)
	defer display.Close()

	for !download.Completed() {
	}
}

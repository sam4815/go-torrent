package main

import (
	"log"
	"os"
	"time"

	"go-torrent/utils"
)

func main() {
	file, err := os.Open(utils.GetFilePath())
	if err != nil {
		log.Fatal("Error opening file: ", err)
	}
	defer file.Close()

	torrent, err := utils.DecodeBencodedFile(file)
	if err != nil {
		log.Fatal("Error decoding file: ", err)
	}

	download, err := utils.StartDownload(torrent)
	if err != nil {
		log.Fatal("Error initiating download: ", err)
	}
	defer download.Close()

	trackers := utils.NewTrackers(torrent.AnnounceList)
	trackers.Announce(torrent, download)

	display := utils.StartDisplay(download, time.Millisecond*100)
	defer display.Close()

	<-download.Completed
}

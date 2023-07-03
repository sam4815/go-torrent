package main

import (
	"flag"
	"fmt"
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
		log.Fatal(err)
	}
	defer file.Close()

	torrent, err := utils.DecodeBencodedFile(file)
	if err != nil {
		log.Fatal(err)
	}

	trackers := utils.NewTrackers(torrent.AnnounceList)

	peers, err := trackers.Announce(torrent)
	if err != nil {
		log.Fatal(err)
	}

	download, err := utils.StartDownload(peers, torrent)
	if err != nil {
		log.Fatal(err)
	}
	defer download.Close()

	for !download.Completed() {
		time.Sleep(time.Second * 1)
		fmt.Printf("\r%s", download.Progress())
	}

	err = download.WriteFiles()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\r%s\n", download.Progress())
}

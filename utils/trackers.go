package utils

import (
	"net/url"
	"sync"
)

type Trackers []Tracker

func NewTrackers(announceList []string) Trackers {
	trackers := make([]Tracker, 0)

	for _, announceURLString := range announceList {
		announceUrl, err := url.Parse(announceURLString)
		if err != nil {
			continue
		}

		tracker := Tracker{AnnounceURL: announceUrl}
		trackers = append(trackers, tracker)
	}

	return trackers
}

func (trackers Trackers) Announce(torrent TorrentFile, download *Download) {
	peersMap := make(map[string]Peer)
	peersMapLock := sync.Mutex{}

	for _, tracker := range trackers {
		go func(tracker Tracker, torrent TorrentFile) {
			peers, err := tracker.Announce(torrent)
			if err != nil {
				Debugf("Error connecting to tracker %s: %s", tracker.AnnounceURL, err)
				return
			}

			for _, peer := range peers {
				peersMapLock.Lock()
				peersMap[peer.IP.String()] = peer
				peersMapLock.Unlock()

				go download.AddPeer(peer)
			}
		}(tracker, torrent)
	}
}

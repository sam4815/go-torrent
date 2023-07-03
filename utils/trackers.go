package utils

import (
	"net/url"
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

func (trackers Trackers) Announce(torrent TorrentFile) ([]Peer, error) {
	peersChan := make(chan []Peer, 1)
	peersMap := make(map[string]Peer)

	for _, tracker := range trackers {
		go func(tracker Tracker, torrent TorrentFile) {
			peers, err := tracker.Announce(torrent)
			if err != nil {
				peersChan <- make([]Peer, 0)
				return
			}

			peersChan <- peers
		}(tracker, torrent)
	}

	for range trackers {
		peers := <-peersChan

		for _, peer := range peers {
			peersMap[peer.IP.String()] = peer
		}
	}

	peers := make([]Peer, 0)

	for _, peer := range peersMap {
		peers = append(peers, peer)
	}

	return peers, nil
}

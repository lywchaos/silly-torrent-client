package main

import (
	"log"
	"runtime/debug"
	"testing"
)

func TestTracker(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	torrent_file, err := OpenTorrentFile("./debian-12.4.0-amd64-netinst.iso.torrent")
	if err != nil {
		log.Fatal(err)
	}
	TrackerResponse, err := RequestTracker(&torrent_file, [20]byte{}, 6888)
	if err != nil {
		log.Println(string(debug.Stack()[:]))
		log.Fatal(err)
	}
	log.Println(TrackerResponse.Interval)
	peers, err := GetAllPeers(&TrackerResponse)
	if err != nil {
		log.Fatal(err)
	}
	for _, peer := range peers {
		log.Println(peer)
	}
}

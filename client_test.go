package main

import (
	"log"
	"testing"
)

func TestHandshake(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	tf, _ := OpenTorrentFile("./debian-12.4.0-amd64-netinst.iso.torrent")
	tr, _ := RequestTracker(&tf, [20]byte{}, 6882)
	all_peers, _ := GetAllPeers(&tr)

	for _, peer := range all_peers {
		_, err := NewClient(&tf, [20]byte{}, peer)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

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
		client, err := NewClient(&tf, [20]byte{}, peer)
		if err != nil {
			log.Println(err)
			continue
		}
		peer_id, err := client.Handshake()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("Successfully complete handshake with %s, which has peer_id %s", peer.String(), (peer_id[:]))
	}
}

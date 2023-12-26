package main

import (
	"testing"
)

func TestOpenTorrentFile(t *testing.T) {
	file, err := OpenTorrentFile("./debian-12.4.0-amd64-netinst.iso.torrent")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(file.Announce)
	t.Log(file.Info.Name)
	t.Log(file.Info.Length)
	t.Log(file.Info.PieceLength)
}

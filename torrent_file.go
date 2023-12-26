package main

import (
	"os"

	"github.com/anacrolix/torrent/bencode"
)

type TorrentFile struct {
	Announce string      `bencode:"announce"`
	Info     TorrentInfo `bencode:"info"`
}

type TorrentInfo struct {
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
	Length      int    `bencode:"length"`
}

func OpenTorrentFile(torrent_file string) (TorrentFile, error) {
	file, err := os.ReadFile(torrent_file)
	if err != nil {
		return TorrentFile{}, err
	}

	var torrent TorrentFile
	err = bencode.Unmarshal(file, &torrent)
	if err != nil {
		return TorrentFile{}, err
	}

	return torrent, nil
}

package main

import (
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/anacrolix/torrent/bencode"
)

type TrackerResponse struct {
	Failure  string `bencode:"failure,omitempty"`
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"` // 按 https://www.bittorrent.org/beps/bep_0003.html 这里的 specification 的话，没法 unmarshal，只能先当成 string，再去 ad hoc parse
}

type Peer struct {
	// PeerID string `bencode:"peer id"`
	// 不需要 peer id，按 https://www.bittorrent.org/beps/bep_0023.html 这里所说，compact 模式下缺失 peer id（而正式 specification 里是有说要 peer id 的）已经是 de-facto
	IP   net.IP `bencode:"ip"`
	Port uint16 `bencode:"port"`
}

func (p *Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}

func InfoSha1Sum(torrent *TorrentFile) ([20]byte, error) {
	info_bytes, err := bencode.Marshal(torrent.Info)
	if err != nil {
		return [20]byte{}, err
	}
	sum := sha1.Sum(info_bytes)
	return sum, nil
}

func RequestTracker(torrent *TorrentFile, peerID [20]byte, port uint16) (TrackerResponse, error) {
	sum, err := InfoSha1Sum(torrent)
	if err != nil {
		return TrackerResponse{}, err
	}
	params := url.Values{
		"info_hash":  []string{string(sum[:])}, // []bytes 转 string 的惯例写法
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"}, // https://www.bittorrent.org/beps/bep_0023.html，以 compact 形式请求，此时结果的 peers 不是一个 list，而是一个 packed string
		"left":       []string{strconv.Itoa(torrent.Info.Length)},
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	tracker_base_url, err := url.Parse(torrent.Announce)
	if err != nil {
		return TrackerResponse{}, err
	}
	tracker_base_url.RawQuery = params.Encode() // 组装
	resp, err := client.Get(tracker_base_url.String())
	if err != nil {
		return TrackerResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TrackerResponse{}, err
	}
	res := TrackerResponse{}
	err = bencode.Unmarshal(body, &res)
	if err != nil {
		log.Println(err)
		return TrackerResponse{}, err
	} else if len(res.Failure) != 0 {
		log.Fatal(res.Failure)
	}
	return res, nil
}

func GetAllPeers(tr *TrackerResponse) ([]Peer, error) {
	peers_bytes := []byte(tr.Peers) // string to []byte
	bytes_per_peer := 6             // ref: https://www.bittorrent.org/beps/bep_0023.html，前 4 个 byte 是 ipv4 address，后 2 个 byte 是 port
	bytes_per_ip := 4
	bytes_per_port := 2
	if len(peers_bytes)%bytes_per_peer != 0 {
		log.Printf("malformed, the length of compact peers string must be a integer multiple of 6, but got length %v", len(peers_bytes))
		return []Peer{}, errors.New("malformed peers string")
	}
	res := make([]Peer, 0)
	for i := 0; i < len(peers_bytes); i += bytes_per_peer {
		res = append(res, Peer{
			IP:   net.IP(peers_bytes[i : i+bytes_per_ip]),
			Port: binary.BigEndian.Uint16(peers_bytes[i+bytes_per_ip : i+bytes_per_ip+bytes_per_port]),
		})
	}
	return res, nil
}

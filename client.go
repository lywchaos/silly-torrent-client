package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"time"
)

type Client struct {
	Conn       net.Conn
	Choked     bool
	Interested bool
	Torrent    *TorrentFile
	PeerID     [20]byte
}

func NewClient(tf *TorrentFile, id [20]byte, peer Peer) (*Client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 15*time.Duration(time.Second))
	if err != nil {
		return nil, err
	}
	return &Client{
		Conn:       conn,
		Choked:     true,
		Interested: false,
		Torrent:    tf,
		PeerID:     id,
	}, nil
}

func (c *Client) Handshake() ([]byte, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// timeout
	err := c.Conn.SetDeadline(time.Now().Add(15 * time.Second))
	if err != nil {
		return nil, err
	}
	defer c.Conn.SetDeadline(time.Time{})

	// construct handshake message
	pstr := "BitTorrent protocol"
	handshake_message := make([]byte, len(pstr)+49)
	handshake_message[0] = byte(len(pstr)) // length prefix of pstr
	cur := 1
	cur += copy(handshake_message[cur:], pstr)            // pstr
	cur += copy(handshake_message[cur:], make([]byte, 8)) // reserved bytes
	sum, err := InfoSha1Sum(c.Torrent)
	if err != nil {
		return make([]byte, 0), err
	}
	cur += copy(handshake_message[cur:], sum[:])      // 20 bytes sha1 hash of the bencoded form of the info value
	cur += copy(handshake_message[cur:], c.PeerID[:]) // 20 bytes peer id

	// send handshake message
	_, err = c.Conn.Write(handshake_message)
	if err != nil {
		return make([]byte, 0), err
	}

	// read and validate handshake message
	res_pstr_len := make([]byte, 1)
	_, err = io.ReadFull(c.Conn, res_pstr_len)
	if err != nil {
		return make([]byte, 0), err
	}
	if int(res_pstr_len[0]) == 0 {
		log.Println("pstr len must not be 0")
		return make([]byte, 0), errors.New("pstr len must not be 0")
	}
	res := make([]byte, len(pstr)+48)
	_, err = io.ReadFull(c.Conn, res)
	if err != nil {
		return make([]byte, 0), err
	}
	info_sum, err := InfoSha1Sum(c.Torrent)
	if err != nil {
		return make([]byte, 0), err
	}
	if !bytes.Equal(res[len(pstr)+8:len(pstr)+28], info_sum[:]) {
		return make([]byte, 0), errors.New("info hash not match")
	}
	peer_id := make([]byte, 20)
	_ = copy(peer_id, res[len(pstr)+28:])

	return peer_id, nil
}

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Message_ID byte

const (
	CHOKE          Message_ID = 0
	UNCHOKE        Message_ID = 1
	INTERESTED     Message_ID = 2
	NOT_INTERESTED Message_ID = 3
	HAVE           Message_ID = 4
	BITFIELD       Message_ID = 5
	REQUEST        Message_ID = 6
	PIECE          Message_ID = 7
	CANCLE         Message_ID = 8
)

type Message struct {
	ID      Message_ID
	Payload []byte
}

func (m *Message) name() string {
	switch m.ID {
	case CHOKE:
		return "CHOKE"
	case UNCHOKE:
		return "UNCHOKE"
	case INTERESTED:
		return "INTERESTED"
	case NOT_INTERESTED:
		return "NOT_INTERESTED"
	case HAVE:
		return "HAVE"
	case BITFIELD:
		return "BITFIELD"
	case REQUEST:
		return "REQUEST"
	case PIECE:
		return "PIECE"
	case CANCLE:
		return "CANCLE"
	default:
		return "UNKNOWN"
	}
}

func readMessage(conn net.Conn) (*Message, error) {
	length_bytes := make([]byte, 4) // all later integer after handshake (not exactly) are encoded as four bytes big-endian
	_, err := io.ReadFull(conn, length_bytes)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(length_bytes) // four bytes, hence uint32

	if length == 0 { // keepalive
		return nil, nil
	}

	message := make([]byte, length)
	_, err = io.ReadFull(conn, message)
	if err != nil {
		return nil, err
	}
	return &Message{
		ID:      Message_ID(message[0]),
		Payload: message[1:],
	}, nil
}

type Client struct {
	Conn       net.Conn
	Choked     bool
	Interested bool
	Torrent    *TorrentFile
	PeerID     [20]byte
	Bitfield   []byte
}

func NewClient(tf *TorrentFile, id [20]byte, peer Peer) (*Client, error) { // communicate with a single peer
	conn, err := net.DialTimeout("tcp", peer.String(), 15*time.Duration(time.Second))
	if err != nil {
		return nil, err
	}
	_client := &Client{
		Conn:       conn,
		Choked:     true,
		Interested: false,
		Torrent:    tf,
		PeerID:     id,
		Bitfield:   make([]byte, 0),
	}

	_, err = _client.handshake()
	if err != nil {
		return nil, err
	}

	message, err := readMessage(_client.Conn)
	if err != nil {
		return nil, err
	}

	if message.ID != BITFIELD {
		return nil, fmt.Errorf("expected BITFIELD as first message, got %s", message.name()) // vscode said that should prefer fmt.Errorf over errors.New(fmt.Sprintf)
	}
	_client.Bitfield = append(_client.Bitfield, message.Payload...) // unpackage in golang

	return _client, nil
}

func (c *Client) handshake() ([]byte, error) {
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

	log.Printf("Successfully complete handshake with %s", (peer_id[:]))
	return peer_id, nil
}

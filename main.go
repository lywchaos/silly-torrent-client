package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"log"
	"math"
	"os"
	"strings"
)

type Piece struct {
	Index  int
	Length int
}

type PieceProgress struct {
	Requested  int
	Downloaded int
	Backlog    int
	Buf        []byte
	Clt        *Client
	Index      int
	Length     int
}

const (
	MaxRequestLength int = 16384 // max length a request message can ask for; value bigger than this will cause server side disconnect
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	torrent_file := os.Args[1]
	output_file := os.Args[2]
	buf, err := download(torrent_file)
	if err != nil {
		log.Fatal(err)
	}
	output_file_handle, err := os.Create(output_file)
	if err != nil {
		log.Fatal(err)
	}
	defer output_file_handle.Close()

	_, err = output_file_handle.Write(buf)
	if err != nil {
		log.Fatal(err)
	}
}

func download(torrent_file_path string) ([]byte, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	tf, _ := OpenTorrentFile(torrent_file_path)
	tr, _ := RequestTracker(&tf, [20]byte{}, 6882)
	all_peers, _ := GetAllPeers(&tr)

	id := [20]byte{}
	rand.Read(id[:])

	num_pieces := int(math.Ceil(float64(tf.Info.Length) / float64(tf.Info.PieceLength)))
	log.Printf("total num pieces is %d", num_pieces)
	jobs := make(chan *Piece, num_pieces)
	for i := 0; i < num_pieces; i++ {
		var length int = tf.Info.PieceLength
		// log.Println(length) right 262144
		if i == num_pieces-1 {
			length = tf.Info.Length - (num_pieces-1)*tf.Info.PieceLength
			log.Printf("last piece length is %d", length)
		}
		jobs <- &Piece{
			Index:  i,
			Length: length,
		}
	}
	tmp_result := make(chan *PieceProgress)

	for _, peer := range all_peers {
		go func(p Peer) {
			client, err := NewClient(&tf, id, p)
			if err != nil {
				log.Println(err)
				return
			}

			client.SendUnchoke()
			client.SendInterested()

			for job := range jobs {
				if !client.CanRequest(job) { // if this peer don't have the piece we want, just put back
					jobs <- job
				}

				pp := PieceProgress{
					Clt:    client,
					Index:  job.Index,
					Buf:    make([]byte, job.Length),
					Length: job.Length,
				}

				// Download
				send_res := true
				for pp.Downloaded < job.Length {
					if !send_res {
						break
					}
					if !pp.Clt.Choked {
						// log.Println("not choked")
						if pp.Backlog < MaxBacklog && pp.Requested < job.Length {
							block_size := MaxRequestLength
							if _block := job.Length - pp.Requested; _block < block_size {
								block_size = _block
							}

							err := pp.Clt.SendRequest(job.Index, pp.Requested, block_size)
							if err != nil {
								log.Println(err.Error())
								if strings.Contains(err.Error(), "broken pipe") {
									jobs <- job
									return
									// send_res = false
									// break
								}
								continue
							}
							if !send_res {
								break
							}
							pp.Backlog++
							pp.Requested += block_size
						}
						if !send_res {
							break
						}
					}
					if !send_res {
						break
					}
					message, err := pp.Clt.readMessage()
					// log.Printf("got message type %s", message.name())
					if err != nil {
						log.Println(err.Error())
						if strings.Contains(err.Error(), "EOF") {
							jobs <- job
							return
							// break
						}
						continue
					}
					if message == nil {
						log.Println("got keepalive message")
						continue
					}
					tmp_res_buf, err := pp.Clt.processMessage(message)
					if err != nil {
						log.Println(err)
						continue
					}
					if tmp_res_buf != nil { // got piece
						tmp_res_begin := binary.BigEndian.Uint32(tmp_res_buf[4:8])
						tmp_res_piece := tmp_res_buf[8:]
						copy(pp.Buf[tmp_res_begin:], tmp_res_piece)
						pp.Downloaded += len(tmp_res_piece)
						pp.Backlog--
					}
				}

				// check sum
				sum1 := sha1.Sum(pp.Buf)
				sum2 := []byte(pp.Clt.Torrent.Info.Pieces[pp.Index*20 : pp.Index*20+20])
				if !bytes.Equal(sum1[:], sum2) {
					// log.Printf("pp requested %d", pp.Requested)
					// log.Printf("pp downloaded %d", pp.Downloaded)
					// log.Printf("pp index %d", pp.Index)
					// log.Printf("pp length %d", pp.Length)
					// log.Printf("pp backlog %d", pp.Backlog)
					// log.Println(sum1)
					// log.Println(sum2)
					log.Println("failed check sum")
					jobs <- job
				} else {
					tmp_result <- &pp
				}
			}
		}(peer)
	}

	// put together
	total_buf := make([]byte, tf.Info.Length)
	num_done_piece := 0
	for num_done_piece < num_pieces {
		done_piece := <-tmp_result
		log.Printf("done_piece requested %d", done_piece.Requested)
		log.Printf("done_piece downloaded %d", done_piece.Downloaded)
		log.Printf("done_piece index %d", done_piece.Index)
		log.Printf("done_piece length %d", done_piece.Length)
		log.Printf("done_piece backlog %d", done_piece.Backlog)
		piece_index, piece_length := done_piece.Index, done_piece.Length
		normal_piece_length := done_piece.Clt.Torrent.Info.PieceLength
		copy(total_buf[piece_index*normal_piece_length:piece_index*normal_piece_length+piece_length], done_piece.Buf)
		num_done_piece++
		// progress bar
		percent := float64(num_done_piece) / float64(len(done_piece.Clt.Torrent.Info.Pieces)/20) * 100
		log.Printf("(%.2f) downloaded index %d ...", percent, done_piece.Index)
	}

	return total_buf, nil
}

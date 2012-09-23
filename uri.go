package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
)

type Magnet struct {
	InfoHashes []string
}

func parseMagnet(s string) (Magnet, error) {
	// References:
	// - http://bittorrent.org/beps/bep_0009.html
	// - http://en.wikipedia.org/wiki/Magnet_URI_scheme
	//
	// Example bittorrent magnet link:
	//
	// => magnet:?xt=urn:btih:bbb6db69965af769f664b6636e7914f8735141b3&dn=Ubuntu-12.04-desktop-i386.iso
	//
	// xt: exact topic.
	//   ~ urn: uniform resource name.
	//   ~ btih: bittorrent infohash. 
	// dn: display name (optional).
	// tr: address tracker (optional).
	u, err := url.Parse(s)
	if err != nil {
		return Magnet{}, err
	}
	xts, ok := u.Query()["xt"]
	if !ok {
		return Magnet{}, fmt.Errorf("Magnet URI missing the 'xt' argument: " + s)
	}
	infoHashes := make([]string, 0, len(xts))
	for _, xt := range xts {
		s := strings.Split(xt, "urn:btih:")
		if len(s) != 2 {
			return Magnet{}, fmt.Errorf("Magnet URI xt parameter missing the 'urn:btih:' prefix. Not a bittorrent hash link?")
		}
		ih := s[1]
		// TODO: support base32 encoded hashes, if they still exist.
		if len(ih) != sha1.Size*2 { // hex format.
			return Magnet{}, fmt.Errorf("Magnet URI contains infohash with unexpected length. Wanted %d, got %d: %v", sha1.Size, len(ih), ih)
		}
		infoHashDecoded, err := hex.DecodeString(ih)
		if err != nil {
			return Magnet{}, fmt.Errorf("Magnet URI contains infohash that cant be hex-decoded '%q': %v", ih, err)
		}
		infoHashes = append(infoHashes, string(infoHashDecoded))
	}
	return Magnet{infoHashes}, nil
}

// torrentFromMagnet fetches the content of a torrent meta file from the magnet
// uri. It only uses the first infohash found in the URI.
func torrentFromMagnet(uri string) (torrentBody io.ReadCloser, err error) {
	m, err := parseMagnet(uri)
	if err != nil {
		return nil, err
	}
	if len(m.InfoHashes) == 0 {
		return nil, fmt.Errorf("No bittorrent infohashes found in the magnet link %v.", uri)
	}
	return magnetDownload(m.InfoHashes[0])
	// Start a torrent session to download the magnet file.
	//
	// References:
	// - http://bittorrent.org/beps/bep_0009.html
	// TODO: Refactor the torrent code to support multiple sessions per client.
}

func magnetDownload(ih string) (torrentBody io.ReadCloser, err error) {
	log.Println("XXX2", ih)
	infoDict := InfoDict{
		Name: "magnetfile.torrent",
		// Single File Mode
		PieceLength: 16384, // BEP 0009
		Md5sum:      ih,
		// Multiple File mode
	}
	t := &TorrentSession{
		m:               &MetaInfo{InfoHash: ih, Info: infoDict},
		peers:           make(map[string]*peerState),
		peerMessageChan: make(chan peerMessage),
		activePieces:    make(map[int]*ActivePiece),
	}

	if err := t.start(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("Not supported. Would have downloaded torrent file with hash %v", ih)
}

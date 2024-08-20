package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

type Peer string

const (
	PeerSender   Peer = "sender"
	PeerReceiver Peer = "receiver"
)

// InvalidPeerErr means peer could not be verified due to missing or malformed peer id.
var ErrInvalidPeer = errors.New("invalid peer id")

func (p Peer) Pid(id string) string {
	data := id + ":" + string(p)
	hash := hmac.New(sha256.New, secret)
	hash.Write([]byte(data))
	signature := hash.Sum(nil)
	return base64.URLEncoding.EncodeToString(signature) + "." + base64.URLEncoding.EncodeToString([]byte(data))
}

func ParsePeer(id string, pid string) (Peer, error) {
	pidParts := strings.SplitN(pid, ".", 2)
	if len(pidParts) < 2 {
		return "", ErrInvalidPeer
	}
	base64Signature, base64Data := pidParts[0], pidParts[1]
	signatureBytes, err := base64.URLEncoding.DecodeString(base64Signature)
	if err != nil {
		return "", ErrInvalidPeer
	}
	dataBytes, err := base64.URLEncoding.DecodeString(base64Data)
	if err != nil {
		return "", ErrInvalidPeer
	}

	// verify signature
	hash := hmac.New(sha256.New, secret)
	hash.Write(dataBytes)
	verifySignature := hash.Sum(nil)
	if !hmac.Equal(signatureBytes, verifySignature) {
		return "", ErrInvalidPeer
	}

	// verify id is same and peer value is valid
	data := string(dataBytes)
	dataParts := strings.SplitN(data, ":", 2)
	if len(dataParts) < 2 {
		return "", ErrInvalidPeer
	}
	dataId, dataPeer := dataParts[0], Peer(dataParts[1])
	if dataId != id || (dataPeer != PeerSender && dataPeer != PeerReceiver) {
		return "", ErrInvalidPeer
	}
	return dataPeer, nil
}

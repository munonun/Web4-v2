package protocol

import (
	"fmt"

	web4crypto "web4/crypto"
)

func NewGETMessage(timestamp, ttl int64, transferID web4crypto.Hash) Message {
	return newMessage(timestamp, ttl, GET, EncodeGETID(transferID))
}

func EncodeGETID(transferID web4crypto.Hash) []byte {
	buf := make([]byte, len(transferID))
	copy(buf, transferID[:])
	return buf
}

func DecodeGETID(payload []byte) (web4crypto.Hash, error) {
	if len(payload) != len(web4crypto.Hash{}) {
		return web4crypto.Hash{}, fmt.Errorf("invalid GET payload length: %d", len(payload))
	}

	var transferID web4crypto.Hash
	copy(transferID[:], payload)
	return transferID, nil
}

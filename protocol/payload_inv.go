package protocol

import (
	"fmt"

	web4crypto "web4/crypto"
)

func NewINVMessage(timestamp, ttl int64, transferID web4crypto.Hash) Message {
	return newMessage(timestamp, ttl, INV, EncodeINVID(transferID))
}

func EncodeINVID(transferID web4crypto.Hash) []byte {
	buf := make([]byte, len(transferID))
	copy(buf, transferID[:])
	return buf
}

func DecodeINVID(payload []byte) (web4crypto.Hash, error) {
	if len(payload) != len(web4crypto.Hash{}) {
		return web4crypto.Hash{}, fmt.Errorf("invalid INV payload length: %d", len(payload))
	}

	var transferID web4crypto.Hash
	copy(transferID[:], payload)
	return transferID, nil
}

func newMessage(timestamp, ttl int64, msgType MsgType, payload []byte) Message {
	return Message{
		MessageID: web4crypto.GenerateMessageID(),
		Timestamp: timestamp,
		TTL:       ttl,
		Type:      msgType,
		Payload:   payload,
	}
}

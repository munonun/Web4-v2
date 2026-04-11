package protocol

import "fmt"

func NewACKMessage(timestamp, ttl int64, referencedMessageID [16]byte) Message {
	return newMessage(timestamp, ttl, ACK, EncodeACKPayload(referencedMessageID))
}

func EncodeACKPayload(referencedMessageID [16]byte) []byte {
	buf := make([]byte, len(referencedMessageID))
	copy(buf, referencedMessageID[:])
	return buf
}

func DecodeACKPayload(payload []byte) ([16]byte, error) {
	if len(payload) != 16 {
		return [16]byte{}, fmt.Errorf("invalid ACK payload length: %d", len(payload))
	}

	var referencedMessageID [16]byte
	copy(referencedMessageID[:], payload)
	return referencedMessageID, nil
}

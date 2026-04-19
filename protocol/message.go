package protocol

import (
	"encoding/binary"
	"fmt"
)

type MsgType int

const (
	INV MsgType = iota
	GET
	TX
	PROOF_REQ
	PROOF_RESP
	ACK
	ERROR
)

type Message struct {
	MessageID [16]byte
	Timestamp int64
	TTL       int64
	Type      MsgType
	Payload   []byte
}

func EncodeMessage(m Message) []byte {
	buf := make([]byte, 0, encodedMessageLen(m))
	buf = append(buf, m.MessageID[:]...)
	buf = appendInt64(buf, m.Timestamp)
	buf = appendInt64(buf, m.TTL)
	buf = appendUint32(buf, uint32(m.Type))
	return appendBytes(buf, m.Payload)
}

func DecodeMessage(payload []byte) (Message, error) {
	const minMessageLen = 16 + 8 + 8 + 4 + 4
	if len(payload) < minMessageLen {
		return Message{}, fmt.Errorf("invalid message length: %d", len(payload))
	}

	var msg Message
	copy(msg.MessageID[:], payload[:16])
	msg.Timestamp = int64(binary.BigEndian.Uint64(payload[16:24]))
	msg.TTL = int64(binary.BigEndian.Uint64(payload[24:32]))

	typ := decodeUint32(payload[32:36])
	if typ > uint32(ERROR) {
		return Message{}, fmt.Errorf("invalid message type: %d", typ)
	}
	msg.Type = MsgType(typ)

	decodedPayload, offset, err := decodeLengthPrefixedBytes(payload, 36)
	if err != nil {
		return Message{}, err
	}
	if offset != len(payload) {
		return Message{}, fmt.Errorf("invalid message length: %d", len(payload))
	}
	msg.Payload = decodedPayload
	return msg, nil
}

func encodedMessageLen(m Message) int {
	return len(m.MessageID) + 8 + 8 + 4 + 4 + len(m.Payload)
}

func appendBytes(dst, value []byte) []byte {
	dst = appendUint32(dst, uint32(len(value)))
	return append(dst, value...)
}

func appendUint32(dst []byte, value uint32) []byte {
	return binary.BigEndian.AppendUint32(dst, value)
}

func appendInt64(dst []byte, value int64) []byte {
	return binary.BigEndian.AppendUint64(dst, uint64(value))
}

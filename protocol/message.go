package protocol

import "encoding/binary"

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

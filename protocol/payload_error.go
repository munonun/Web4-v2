package protocol

import "fmt"

type ErrorPayload struct {
	ReferencedMessageID [16]byte
	Code                string
}

func NewErrorMessage(timestamp, ttl int64, referencedMessageID [16]byte, code string) Message {
	return newMessage(timestamp, ttl, ERROR, EncodeErrorPayload(referencedMessageID, code))
}

func EncodeErrorPayload(referencedMessageID [16]byte, code string) []byte {
	buf := make([]byte, 0, len(referencedMessageID)+4+len(code))
	buf = append(buf, referencedMessageID[:]...)
	return appendBytes(buf, []byte(code))
}

func DecodeErrorPayload(payload []byte) (ErrorPayload, error) {
	if len(payload) < 20 {
		return ErrorPayload{}, fmt.Errorf("invalid ERROR payload length: %d", len(payload))
	}

	var decoded ErrorPayload
	copy(decoded.ReferencedMessageID[:], payload[:16])

	textLen := int(decodeUint32(payload[16:20]))
	if len(payload) != 20+textLen {
		return ErrorPayload{}, fmt.Errorf("invalid ERROR payload length: %d", len(payload))
	}
	decoded.Code = string(payload[20:])
	return decoded, nil
}

func decodeUint32(payload []byte) uint32 {
	return uint32(payload[0])<<24 |
		uint32(payload[1])<<16 |
		uint32(payload[2])<<8 |
		uint32(payload[3])
}

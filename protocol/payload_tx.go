package protocol

import (
	"fmt"

	web4crypto "web4/crypto"
)

type TXPayload struct {
	TransferBytes []byte
	Signature     []byte
}

func NewTXMessage(timestamp, ttl int64, transfer web4crypto.Transfer) Message {
	return newMessage(timestamp, ttl, TX, EncodeTXPayload(web4crypto.EncodeTransfer(transfer), transfer.Sig))
}

func EncodeTXPayload(transferBytes, signature []byte) []byte {
	buf := make([]byte, 0, 4+len(transferBytes)+4+len(signature))
	buf = appendBytes(buf, transferBytes)
	return appendBytes(buf, signature)
}

func DecodeTXPayload(payload []byte) (TXPayload, error) {
	transferBytes, offset, err := decodeLengthPrefixedBytes(payload, 0)
	if err != nil {
		return TXPayload{}, err
	}
	signature, offset, err := decodeLengthPrefixedBytes(payload, offset)
	if err != nil {
		return TXPayload{}, err
	}
	if offset != len(payload) {
		return TXPayload{}, fmt.Errorf("invalid TX payload length: %d", len(payload))
	}

	return TXPayload{TransferBytes: transferBytes, Signature: signature}, nil
}

func decodeLengthPrefixedBytes(payload []byte, offset int) ([]byte, int, error) {
	if len(payload[offset:]) < 4 {
		return nil, offset, fmt.Errorf("short length prefix at offset %d", offset)
	}
	length := int(decodeUint32(payload[offset : offset+4]))
	offset += 4
	if len(payload[offset:]) < length {
		return nil, offset, fmt.Errorf("short length-prefixed data at offset %d", offset)
	}
	decoded := make([]byte, length)
	copy(decoded, payload[offset:offset+length])
	return decoded, offset + length, nil
}

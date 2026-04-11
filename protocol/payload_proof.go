package protocol

import (
	"fmt"

	web4crypto "web4/crypto"
)

type ProofResponsePayload struct {
	TargetID web4crypto.Hash
	Seen     bool
	SeenTime int64
	Conflict bool
	Distance int
}

func NewProofRequestMessage(timestamp, ttl int64, transferID web4crypto.Hash) Message {
	return newMessage(timestamp, ttl, PROOF_REQ, EncodeProofRequestPayload(transferID))
}

func EncodeProofRequestPayload(transferID web4crypto.Hash) []byte {
	buf := make([]byte, len(transferID))
	copy(buf, transferID[:])
	return buf
}

func DecodeProofRequestPayload(payload []byte) (web4crypto.Hash, error) {
	if len(payload) != len(web4crypto.Hash{}) {
		return web4crypto.Hash{}, fmt.Errorf("invalid PROOF_REQ payload length: %d", len(payload))
	}

	var transferID web4crypto.Hash
	copy(transferID[:], payload)
	return transferID, nil
}

func NewProofResponseMessage(timestamp, ttl int64, payload ProofResponsePayload) Message {
	return newMessage(timestamp, ttl, PROOF_RESP, EncodeProofResponsePayload(payload))
}

func EncodeProofResponsePayload(payload ProofResponsePayload) []byte {
	buf := make([]byte, 0, len(payload.TargetID)+1+8+1+8)
	buf = append(buf, payload.TargetID[:]...)
	buf = appendBool(buf, payload.Seen)
	buf = appendInt64(buf, payload.SeenTime)
	buf = appendBool(buf, payload.Conflict)
	return appendInt64(buf, int64(payload.Distance))
}

func DecodeProofResponsePayload(payload []byte) (ProofResponsePayload, error) {
	const payloadLen = 32 + 1 + 8 + 1 + 8
	if len(payload) != payloadLen {
		return ProofResponsePayload{}, fmt.Errorf("invalid PROOF_RESP payload length: %d", len(payload))
	}

	var decoded ProofResponsePayload
	copy(decoded.TargetID[:], payload[:32])

	seen, err := decodeBool(payload[32])
	if err != nil {
		return ProofResponsePayload{}, err
	}
	decoded.Seen = seen
	decoded.SeenTime = int64(decodeUint64(payload[33:41]))

	conflict, err := decodeBool(payload[41])
	if err != nil {
		return ProofResponsePayload{}, err
	}
	decoded.Conflict = conflict

	distance := int64(decodeUint64(payload[42:50]))
	decoded.Distance = int(distance)
	if int64(decoded.Distance) != distance {
		return ProofResponsePayload{}, fmt.Errorf("invalid PROOF_RESP distance: %d", distance)
	}

	return decoded, nil
}

func appendBool(dst []byte, value bool) []byte {
	if value {
		return append(dst, 1)
	}
	return append(dst, 0)
}

func decodeBool(value byte) (bool, error) {
	switch value {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("invalid boolean value: %d", value)
	}
}

func decodeUint64(payload []byte) uint64 {
	return uint64(payload[0])<<56 |
		uint64(payload[1])<<48 |
		uint64(payload[2])<<40 |
		uint64(payload[3])<<32 |
		uint64(payload[4])<<24 |
		uint64(payload[5])<<16 |
		uint64(payload[6])<<8 |
		uint64(payload[7])
}

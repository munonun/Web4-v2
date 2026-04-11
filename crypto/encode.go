package crypto

import "encoding/binary"

func EncodeValue(v Value) []byte {
	buf := make([]byte, 0, encodedValueLen(v))
	return appendValueEncoding(buf, v)
}

// EncodeTransfer preserves the caller-provided input and output order.
// In Web4 that order is semantically meaningful and part of the canonical payload.
func EncodeTransfer(t Transfer) []byte {
	buf := make([]byte, 0, encodedTransferLen(t))
	return appendTransferEncoding(buf, t)
}

func encodedValueLen(v Value) int {
	return 8 + 4 + len(v.Unit) + 4 + len(v.Owner) + 8 + 8
}

func encodedTransferLen(t Transfer) int {
	length := 4 + (len(t.Inputs) * len(Hash{})) + 4
	for _, out := range t.Outputs {
		length += encodedValueLen(out)
	}
	return length
}

func appendTransferEncoding(dst []byte, t Transfer) []byte {
	dst = appendUint32(dst, uint32(len(t.Inputs)))
	for _, input := range t.Inputs {
		dst = append(dst, input[:]...)
	}

	dst = appendUint32(dst, uint32(len(t.Outputs)))
	for _, out := range t.Outputs {
		dst = appendValueEncoding(dst, out)
	}

	return dst
}

func appendValueEncoding(dst []byte, v Value) []byte {
	dst = appendUint64(dst, v.Amount)
	dst = appendBytes(dst, []byte(v.Unit))
	dst = appendBytes(dst, v.Owner)
	dst = appendInt64(dst, v.Expiry)
	return appendInt64(dst, int64(v.Depth))
}

func appendBytes(dst, value []byte) []byte {
	dst = appendUint32(dst, uint32(len(value)))
	return append(dst, value...)
}

func appendUint32(dst []byte, value uint32) []byte {
	return binary.BigEndian.AppendUint32(dst, value)
}

func appendUint64(dst []byte, value uint64) []byte {
	return binary.BigEndian.AppendUint64(dst, value)
}

func appendInt64(dst []byte, value int64) []byte {
	return binary.BigEndian.AppendUint64(dst, uint64(value))
}

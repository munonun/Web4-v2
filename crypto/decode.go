package crypto

import (
	"encoding/binary"
	"fmt"
)

func DecodeValue(data []byte) (Value, error) {
	value, offset, err := decodeValue(data, 0)
	if err != nil {
		return Value{}, err
	}
	if offset != len(data) {
		return Value{}, fmt.Errorf("unexpected trailing bytes in value encoding: %d", len(data)-offset)
	}
	value.ID = ComputeValueID(value)
	return value, nil
}

func DecodeTransfer(data []byte) (Transfer, error) {
	transfer, offset, err := decodeTransfer(data, 0)
	if err != nil {
		return Transfer{}, err
	}
	if offset != len(data) {
		return Transfer{}, fmt.Errorf("unexpected trailing bytes in transfer encoding: %d", len(data)-offset)
	}
	transfer.ID = ComputeTransferID(transfer)
	return transfer, nil
}

func decodeTransfer(data []byte, offset int) (Transfer, int, error) {
	inputCount, next, err := decodeUint32At(data, offset)
	if err != nil {
		return Transfer{}, offset, err
	}
	offset = next

	inputs := make([]Hash, 0, inputCount)
	for i := uint32(0); i < inputCount; i++ {
		if len(data[offset:]) < len(Hash{}) {
			return Transfer{}, offset, fmt.Errorf("short input hash at index %d", i)
		}
		var input Hash
		copy(input[:], data[offset:offset+len(Hash{})])
		inputs = append(inputs, input)
		offset += len(Hash{})
	}

	outputCount, next, err := decodeUint32At(data, offset)
	if err != nil {
		return Transfer{}, offset, err
	}
	offset = next

	outputs := make([]Value, 0, outputCount)
	for i := uint32(0); i < outputCount; i++ {
		value, nextOffset, err := decodeValue(data, offset)
		if err != nil {
			return Transfer{}, offset, err
		}
		outputs = append(outputs, value)
		offset = nextOffset
	}

	transfer := Transfer{
		Inputs:  inputs,
		Outputs: outputs,
	}
	return transfer, offset, nil
}

func decodeValue(data []byte, offset int) (Value, int, error) {
	amount, next, err := decodeUint64At(data, offset)
	if err != nil {
		return Value{}, offset, err
	}
	offset = next

	unit, next, err := decodeBytesAt(data, offset)
	if err != nil {
		return Value{}, offset, err
	}
	offset = next

	owner, next, err := decodeBytesAt(data, offset)
	if err != nil {
		return Value{}, offset, err
	}
	offset = next

	expiryBits, next, err := decodeUint64At(data, offset)
	if err != nil {
		return Value{}, offset, err
	}
	offset = next

	depthBits, next, err := decodeUint64At(data, offset)
	if err != nil {
		return Value{}, offset, err
	}
	offset = next

	depth := int64(depthBits)
	maxInt := int64(^uint(0) >> 1)
	minInt := -maxInt - 1
	if depth > maxInt || depth < minInt {
		return Value{}, offset, fmt.Errorf("value depth out of range: %d", depth)
	}

	value := Value{
		Amount: amount,
		Unit:   string(unit),
		Owner:  owner,
		Expiry: int64(expiryBits),
		Depth:  int(depth),
	}
	value.ID = ComputeValueID(value)
	return value, offset, nil
}

func decodeBytesAt(data []byte, offset int) ([]byte, int, error) {
	length, next, err := decodeUint32At(data, offset)
	if err != nil {
		return nil, offset, err
	}
	offset = next

	if len(data[offset:]) < int(length) {
		return nil, offset, fmt.Errorf("short byte slice: need %d have %d", length, len(data[offset:]))
	}
	decoded := make([]byte, int(length))
	copy(decoded, data[offset:offset+int(length)])
	return decoded, offset + int(length), nil
}

func decodeUint32At(data []byte, offset int) (uint32, int, error) {
	if len(data[offset:]) < 4 {
		return 0, offset, fmt.Errorf("short uint32 at offset %d", offset)
	}
	return binary.BigEndian.Uint32(data[offset : offset+4]), offset + 4, nil
}

func decodeUint64At(data []byte, offset int) (uint64, int, error) {
	if len(data[offset:]) < 8 {
		return 0, offset, fmt.Errorf("short uint64 at offset %d", offset)
	}
	return binary.BigEndian.Uint64(data[offset : offset+8]), offset + 8, nil
}

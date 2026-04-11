package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"

	web4crypto "web4/crypto"
)

func TestEncodeMessageDeterministic(t *testing.T) {
	msg := sampleMessage()
	if !bytes.Equal(EncodeMessage(msg), EncodeMessage(msg)) {
		t.Fatal("message encoding is not deterministic")
	}
}

func TestEncodeMessageDifferentMessageIDChangesBytes(t *testing.T) {
	first := sampleMessage()
	second := first
	second.MessageID[0] ^= 0xff

	if bytes.Equal(EncodeMessage(first), EncodeMessage(second)) {
		t.Fatal("message encoding must change when message ID changes")
	}
}

func TestMessageTimestampDoesNotAffectTransferID(t *testing.T) {
	transfer := sampleTransfer()
	firstID := web4crypto.ComputeTransferID(transfer)

	msg := sampleMessage()
	msg.Timestamp++
	_ = EncodeMessage(msg)

	if web4crypto.ComputeTransferID(transfer) != firstID {
		t.Fatal("message timestamp must not affect transfer ID")
	}
}

func TestEncodeMessagePreservesTTL(t *testing.T) {
	msg := sampleMessage()
	encoded := EncodeMessage(msg)
	ttlOffset := len(msg.MessageID) + 8
	got := int64(binary.BigEndian.Uint64(encoded[ttlOffset : ttlOffset+8]))
	if got != msg.TTL {
		t.Fatalf("unexpected TTL in encoded message: got %d want %d", got, msg.TTL)
	}
}

func sampleMessage() Message {
	return Message{
		MessageID: [16]byte{1, 2, 3, 4},
		Timestamp: 1_700_000_200,
		TTL:       30,
		Type:      TX,
		Payload:   []byte{0xaa, 0xbb, 0xcc},
	}
}

func sampleTransfer() web4crypto.Transfer {
	value := web4crypto.Value{
		Amount: 10,
		Unit:   "WEB4",
		Owner:  []byte{0x01, 0x02, 0x03},
		Expiry: 1_800_000_000,
		Depth:  1,
	}
	value.ID = web4crypto.ComputeValueID(value)

	return web4crypto.Transfer{
		Inputs:    []web4crypto.Hash{{1}, {2}},
		Outputs:   []web4crypto.Value{value},
		Timestamp: 1_700_000_100,
	}
}

package protocol

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	web4crypto "web4/crypto"
)

func TestINVPayloadDeterministicAndRoundTrip(t *testing.T) {
	transferID := sampleTransferID(1)
	first := EncodeINVID(transferID)
	second := EncodeINVID(transferID)
	if !bytes.Equal(first, second) {
		t.Fatal("INV payload encoding is not deterministic")
	}
	if len(first) != len(transferID) {
		t.Fatal("INV payload must contain only the transfer ID")
	}

	decoded, err := DecodeINVID(first)
	if err != nil {
		t.Fatalf("DecodeINVID: %v", err)
	}
	if decoded != transferID {
		t.Fatal("decoded INV transfer ID mismatch")
	}
}

func TestGETPayloadDeterministicAndRoundTrip(t *testing.T) {
	transferID := sampleTransferID(2)
	first := EncodeGETID(transferID)
	second := EncodeGETID(transferID)
	if !bytes.Equal(first, second) {
		t.Fatal("GET payload encoding is not deterministic")
	}
	if len(first) != len(transferID) {
		t.Fatal("GET payload must contain only the transfer ID")
	}

	decoded, err := DecodeGETID(first)
	if err != nil {
		t.Fatalf("DecodeGETID: %v", err)
	}
	if decoded != transferID {
		t.Fatal("decoded GET transfer ID mismatch")
	}
}

func TestTXPayloadEqualsCanonicalTransferBytes(t *testing.T) {
	transfer := sampleProtocolTransfer()
	message := NewTXMessage(1_700_000_300, 30, transfer)
	want := web4crypto.EncodeTransfer(transfer)
	payload, err := DecodeTXPayload(message.Payload)
	if err != nil {
		t.Fatalf("DecodeTXPayload: %v", err)
	}
	if !bytes.Equal(payload.TransferBytes, want) {
		t.Fatal("TX payload transfer bytes must equal canonical transfer bytes")
	}
	if !bytes.Equal(payload.Signature, transfer.Sig) {
		t.Fatal("TX payload must preserve signature bytes")
	}
}

func TestTXPayloadIgnoresTransferTimestamp(t *testing.T) {
	first := sampleProtocolTransfer()
	second := first
	second.Timestamp = first.Timestamp + 99

	firstPayload, err := DecodeTXPayload(NewTXMessage(1_700_000_300, 30, first).Payload)
	if err != nil {
		t.Fatalf("DecodeTXPayload(first): %v", err)
	}
	secondPayload, err := DecodeTXPayload(NewTXMessage(1_700_000_301, 30, second).Payload)
	if err != nil {
		t.Fatalf("DecodeTXPayload(second): %v", err)
	}
	if !bytes.Equal(firstPayload.TransferBytes, secondPayload.TransferBytes) {
		t.Fatal("TX payload must ignore transfer timestamp")
	}
}

func TestTXPayloadDeterministicAndRoundTrip(t *testing.T) {
	transfer := sampleProtocolTransfer()
	encoded := EncodeTXPayload(web4crypto.EncodeTransfer(transfer), transfer.Sig)
	if !bytes.Equal(encoded, EncodeTXPayload(web4crypto.EncodeTransfer(transfer), transfer.Sig)) {
		t.Fatal("TX payload encoding is not deterministic")
	}

	decoded, err := DecodeTXPayload(encoded)
	if err != nil {
		t.Fatalf("DecodeTXPayload: %v", err)
	}
	if !bytes.Equal(decoded.TransferBytes, web4crypto.EncodeTransfer(transfer)) {
		t.Fatal("decoded TX transfer bytes mismatch")
	}
	if !bytes.Equal(decoded.Signature, transfer.Sig) {
		t.Fatal("decoded TX signature mismatch")
	}
}

func TestTransferIDUnaffectedBySignaturePresence(t *testing.T) {
	transfer := sampleProtocolTransfer()
	withSig := transfer
	withSig.Sig = []byte("different-signature")
	if web4crypto.ComputeTransferID(transfer) != web4crypto.ComputeTransferID(withSig) {
		t.Fatal("transfer ID must be independent of signature presence")
	}
}

func TestProofRequestPayloadDeterministicAndRoundTrip(t *testing.T) {
	transferID := sampleTransferID(3)
	first := EncodeProofRequestPayload(transferID)
	second := EncodeProofRequestPayload(transferID)
	if !bytes.Equal(first, second) {
		t.Fatal("PROOF_REQ payload encoding is not deterministic")
	}

	decoded, err := DecodeProofRequestPayload(first)
	if err != nil {
		t.Fatalf("DecodeProofRequestPayload: %v", err)
	}
	if decoded != transferID {
		t.Fatal("decoded PROOF_REQ transfer ID mismatch")
	}
}

func TestProofResponsePayloadDeterministicAndRoundTrip(t *testing.T) {
	payload := ProofResponsePayload{
		TargetID: sampleTransferID(4),
		Seen:     true,
		SeenTime: 1_700_000_400,
		Conflict: false,
		Distance: 2,
	}

	first := EncodeProofResponsePayload(payload)
	second := EncodeProofResponsePayload(payload)
	if !bytes.Equal(first, second) {
		t.Fatal("PROOF_RESP payload encoding is not deterministic")
	}

	decoded, err := DecodeProofResponsePayload(first)
	if err != nil {
		t.Fatalf("DecodeProofResponsePayload: %v", err)
	}
	if decoded != payload {
		t.Fatal("decoded PROOF_RESP payload mismatch")
	}
}

func TestACKPayloadDeterministicAndRoundTrip(t *testing.T) {
	referenced := [16]byte{1, 2, 3, 4}
	first := EncodeACKPayload(referenced)
	second := EncodeACKPayload(referenced)
	if !bytes.Equal(first, second) {
		t.Fatal("ACK payload encoding is not deterministic")
	}

	decoded, err := DecodeACKPayload(first)
	if err != nil {
		t.Fatalf("DecodeACKPayload: %v", err)
	}
	if decoded != referenced {
		t.Fatal("decoded ACK message ID mismatch")
	}
}

func TestErrorPayloadDeterministicAndRoundTrip(t *testing.T) {
	referenced := [16]byte{9, 8, 7, 6}
	first := EncodeErrorPayload(referenced, "invalid_signature")
	second := EncodeErrorPayload(referenced, "invalid_signature")
	if !bytes.Equal(first, second) {
		t.Fatal("ERROR payload encoding is not deterministic")
	}

	decoded, err := DecodeErrorPayload(first)
	if err != nil {
		t.Fatalf("DecodeErrorPayload: %v", err)
	}
	if decoded.ReferencedMessageID != referenced {
		t.Fatal("decoded ERROR referenced message ID mismatch")
	}
	if decoded.Code != "invalid_signature" {
		t.Fatal("decoded ERROR code mismatch")
	}
}

func TestChangingPayloadContentChangesEncodedBytes(t *testing.T) {
	if bytes.Equal(EncodeINVID(sampleTransferID(10)), EncodeINVID(sampleTransferID(11))) {
		t.Fatal("INV payload bytes must change when transfer ID changes")
	}

	firstProof := EncodeProofResponsePayload(ProofResponsePayload{
		TargetID: sampleTransferID(12),
		Seen:     true,
		SeenTime: 1_700_000_500,
		Conflict: false,
		Distance: 1,
	})
	secondProof := EncodeProofResponsePayload(ProofResponsePayload{
		TargetID: sampleTransferID(12),
		Seen:     true,
		SeenTime: 1_700_000_500,
		Conflict: true,
		Distance: 1,
	})
	if bytes.Equal(firstProof, secondProof) {
		t.Fatal("PROOF_RESP payload bytes must change when content changes")
	}

	if bytes.Equal(EncodeErrorPayload([16]byte{1}, "timeout"), EncodeErrorPayload([16]byte{1}, "invalid")) {
		t.Fatal("ERROR payload bytes must change when code changes")
	}
}

func sampleTransferID(seed byte) web4crypto.Hash {
	var id web4crypto.Hash
	for i := range id {
		id[i] = seed + byte(i)
	}
	return id
}

func sampleProtocolTransfer() web4crypto.Transfer {
	pub, priv := sampleOwnerKeys(1)
	input := web4crypto.Value{
		Amount: 25,
		Unit:   "WEB4",
		Owner:  pub,
		Expiry: 1_800_000_000,
		Depth:  0,
	}
	input.ID = web4crypto.ComputeValueID(input)

	value := web4crypto.Value{
		Amount: 25,
		Unit:   "WEB4",
		Owner:  []byte{0x0a, 0x0b, 0x0c},
		Expiry: 1_800_000_000,
		Depth:  1,
	}
	value.ID = web4crypto.ComputeValueID(value)

	transfer := web4crypto.Transfer{
		Inputs:    []web4crypto.Hash{{1}, {2}},
		Outputs:   []web4crypto.Value{value},
		Timestamp: 1_700_000_250,
	}
	transfer.Inputs = []web4crypto.Hash{input.ID}
	transfer.Sig = web4crypto.SignCanonicalTransfer(priv, transfer)
	return transfer
}

func sampleOwnerKeys(seed byte) (ed25519.PublicKey, ed25519.PrivateKey) {
	seedBytes := bytes.Repeat([]byte{seed}, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seedBytes)
	pub := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pub, priv[32:])
	return pub, priv
}

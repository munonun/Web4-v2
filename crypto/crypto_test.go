package crypto

import (
	"bytes"
	"crypto/ed25519"
	stdsha256 "crypto/sha256"
	"testing"

	"github.com/zeebo/blake3"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestEncodeValueDeterministicAndIgnoresID(t *testing.T) {
	v := sampleValue()
	first := EncodeValue(v)
	second := EncodeValue(v)
	if !bytes.Equal(first, second) {
		t.Fatal("value encoding is not deterministic")
	}

	v.ID = Hash{1, 2, 3}
	third := EncodeValue(v)
	if !bytes.Equal(first, third) {
		t.Fatal("value encoding must ignore derived ID field")
	}
}

func TestEncodeTransferDeterministicAndIgnoresIDAndSignature(t *testing.T) {
	tr := sampleTransfer()
	first := EncodeTransfer(tr)
	second := EncodeTransfer(tr)
	if !bytes.Equal(first, second) {
		t.Fatal("transfer encoding is not deterministic")
	}

	tr.ID = Hash{9, 8, 7}
	tr.Sig = []byte("signature")
	third := EncodeTransfer(tr)
	if !bytes.Equal(first, third) {
		t.Fatal("transfer encoding must ignore derived ID and signature fields")
	}
}

func TestEncodeTransferIgnoresTimestamp(t *testing.T) {
	first := sampleTransfer()
	second := first
	second.Timestamp = first.Timestamp + 999

	if !bytes.Equal(EncodeTransfer(first), EncodeTransfer(second)) {
		t.Fatal("transfer encoding must ignore timestamp")
	}
}

func TestComputeValueIDIgnoresID(t *testing.T) {
	v := sampleValue()
	want := ComputeValueID(v)
	v.ID = Hash{4, 5, 6}
	got := ComputeValueID(v)
	if got != want {
		t.Fatal("value ID changed when only ID field changed")
	}
}

func TestComputeTransferIDIgnoresIDAndSignature(t *testing.T) {
	tr := sampleTransfer()
	want := ComputeTransferID(tr)
	tr.ID = Hash{4, 5, 6}
	tr.Sig = []byte("signature")
	got := ComputeTransferID(tr)
	if got != want {
		t.Fatal("transfer ID changed when only ID or signature changed")
	}
}

func TestComputeTransferIDIgnoresTimestamp(t *testing.T) {
	first := sampleTransfer()
	second := first
	second.Timestamp = first.Timestamp + 999

	if ComputeTransferID(first) != ComputeTransferID(second) {
		t.Fatal("transfer ID must ignore timestamp")
	}
}

func TestComputeTransferIDDiffersForDifferentOutputs(t *testing.T) {
	first := sampleTransfer()
	second := sampleTransfer()
	second.Outputs[0].Amount++

	if ComputeTransferID(first) == ComputeTransferID(second) {
		t.Fatal("transfer ID must change when outputs change")
	}
}

func TestTransferInputOrderIsSemantic(t *testing.T) {
	first := sampleTransferWithTwoOutputs()
	second := first
	second.Inputs = []Hash{first.Inputs[1], first.Inputs[0]}

	if bytes.Equal(EncodeTransfer(first), EncodeTransfer(second)) {
		t.Fatal("transfer encoding must preserve input order")
	}
	if ComputeTransferID(first) == ComputeTransferID(second) {
		t.Fatal("transfer ID must change when input order changes")
	}
}

func TestTransferOutputOrderIsSemantic(t *testing.T) {
	first := sampleTransferWithTwoOutputs()
	second := first
	second.Outputs = []Value{first.Outputs[1], first.Outputs[0]}

	if bytes.Equal(EncodeTransfer(first), EncodeTransfer(second)) {
		t.Fatal("transfer encoding must preserve output order")
	}
	if ComputeTransferID(first) == ComputeTransferID(second) {
		t.Fatal("transfer ID must change when output order changes")
	}
}

func TestSignAndVerifyTransfer(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	tr := sampleTransfer()
	msg := EncodeTransfer(tr)
	sig := SignTransfer(priv, msg)
	if !VerifyTransfer(pub, msg, sig) {
		t.Fatal("signature verification failed")
	}

	msg[0] ^= 0xff
	if VerifyTransfer(pub, msg, sig) {
		t.Fatal("signature verification succeeded for tampered message")
	}
}

func TestCanonicalSignatureIgnoresTimestamp(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	first := sampleTransfer()
	second := first
	second.Timestamp = first.Timestamp + 999

	sig := SignCanonicalTransfer(priv, first)
	if !VerifyCanonicalTransfer(pub, first, sig) {
		t.Fatal("canonical signature verification failed for original transfer")
	}
	if !VerifyCanonicalTransfer(pub, second, sig) {
		t.Fatal("canonical signature verification must ignore timestamp")
	}

	other := second
	other.Outputs = append([]Value(nil), second.Outputs...)
	other.Outputs[0].Amount++
	if VerifyCanonicalTransfer(pub, other, sig) {
		t.Fatal("canonical signature verification must fail when transfer content changes")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{0x01}, chacha20poly1305.KeySize)
	nonce := bytes.Repeat([]byte{0x02}, chacha20poly1305.NonceSizeX)
	plaintext := []byte("web4 secret payload")

	ciphertext, err := Encrypt(key, nonce, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(ciphertext, plaintext) {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := Decrypt(key, nonce, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatal("decrypted plaintext mismatch")
	}
}

func TestDecryptRejectsWrongKey(t *testing.T) {
	key := bytes.Repeat([]byte{0x01}, chacha20poly1305.KeySize)
	wrongKey := bytes.Repeat([]byte{0x03}, chacha20poly1305.KeySize)
	nonce := bytes.Repeat([]byte{0x02}, chacha20poly1305.NonceSizeX)

	ciphertext, err := Encrypt(key, nonce, []byte("payload"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if _, err := Decrypt(wrongKey, nonce, ciphertext); err == nil {
		t.Fatal("expected decryption failure with wrong key")
	}
}

func TestGenerateMessageID(t *testing.T) {
	first := GenerateMessageID()
	second := GenerateMessageID()
	if first == ([16]byte{}) {
		t.Fatal("message ID should not be zero")
	}
	if first == second {
		t.Fatal("message IDs should be random and unique in practice")
	}
}

func TestHashersMatchReferenceImplementations(t *testing.T) {
	input := []byte("web4-hash-input")

	if got, want := (Blake3Hasher{}).Sum256(input), blake3.Sum256(input); got != want {
		t.Fatal("BLAKE3 hash mismatch")
	}

	if got, want := (SHA256Hasher{}).Sum256(input), stdsha256.Sum256(input); got != want {
		t.Fatal("SHA-256 hash mismatch")
	}
}

func BenchmarkBlake3Sum256(b *testing.B) {
	h := Blake3Hasher{}
	data := bytes.Repeat([]byte("web4"), 256)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_ = h.Sum256(data)
	}
}

func BenchmarkSHA256Sum256(b *testing.B) {
	h := SHA256Hasher{}
	data := bytes.Repeat([]byte("web4"), 256)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for i := 0; i < b.N; i++ {
		_ = h.Sum256(data)
	}
}

func sampleValue() Value {
	return Value{
		Amount: 42,
		Unit:   "WEB4",
		Owner:  []byte{0xaa, 0xbb, 0xcc},
		Expiry: 1_700_000_000,
		Depth:  2,
	}
}

func sampleTransfer() Transfer {
	output := sampleValue()
	output.ID = ComputeValueID(output)

	return Transfer{
		Inputs: []Hash{
			Hash{1},
			Hash{2},
		},
		Outputs:   []Value{output},
		Timestamp: 1_700_000_123,
	}
}

func sampleTransferWithTwoOutputs() Transfer {
	firstOutput := sampleValue()
	firstOutput.ID = ComputeValueID(firstOutput)

	secondOutput := sampleValue()
	secondOutput.Amount = 7
	secondOutput.Owner = []byte{0x11, 0x22, 0x33}
	secondOutput.ID = ComputeValueID(secondOutput)

	return Transfer{
		Inputs: []Hash{
			Hash{1},
			Hash{2},
		},
		Outputs:   []Value{firstOutput, secondOutput},
		Timestamp: 1_700_000_123,
	}
}

package node

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"testing"

	web4crypto "web4/crypto"
	"web4/protocol"
)

func TestINVGETTXFlow(t *testing.T) {
	a := newTestNode("a", 100)
	b := newTestNode("b", 100)
	a.AddPeer(b)
	b.AddPeer(a)

	input, priv := seedKnownInput(a, b, 1, 25)
	tx := signedTransfer(input, priv, 25, 11)
	if err := a.AcceptLocalTransfer(tx); err != nil {
		t.Fatalf("AcceptLocalTransfer: %v", err)
	}
	txID := web4crypto.ComputeTransferID(tx)

	if !b.hasTransfer(txID) {
		t.Fatal("peer did not receive transfer after INV -> GET -> TX flow")
	}
	if len(b.peerObservations[txID]) != 1 {
		t.Fatal("peer observation was not recorded for received transfer")
	}
}

func TestConflictingTransfersSelectOneLineage(t *testing.T) {
	n := newTestNode("n", 200)
	input, priv := seedKnownInput(n, nil, 1, 50)
	first := signedTransfer(input, priv, 25, 10)
	second := signedTransfer(input, priv, 40, 20)

	firstID := mustReceiveTX(t, n, "peer-a", first)
	secondID := mustReceiveTX(t, n, "peer-b", second)
	inputID := first.Inputs[0]

	selected, ok := n.selectedLineage[inputID]
	if !ok {
		t.Fatal("missing selected lineage for conflicting input")
	}
	if selected.TxID != firstID {
		t.Fatal("first-seen transfer should win deterministic selection")
	}
	if state := n.finalityState[secondID]; state.Status != StatusPending {
		t.Fatal("losing conflicting transfer must remain pending")
	}
}

func TestProofResponseUpdatesFinality(t *testing.T) {
	n := newTestNode("n", 100)
	input, priv := seedKnownInput(n, nil, 5, 100_000)
	tx := signedTransfer(input, priv, 100_000, 30)
	txID := mustReceiveTX(t, n, "peer-a", tx)

	initial := n.finalityState[txID]
	if initial.Status != StatusSelected {
		t.Fatal("transfer should start selected before proof response")
	}

	n.now = func() int64 { return 140 }
	proof := protocol.NewProofResponseMessage(140, 30, protocol.ProofResponsePayload{
		TargetID: txID,
		Seen:     true,
		SeenTime: 100,
		Conflict: false,
		Distance: 1,
	})
	if err := n.OnMessage("peer-b", proof); err != nil {
		t.Fatalf("OnMessage(PROOF_RESP): %v", err)
	}

	updated := n.finalityState[txID]
	if updated.Status != StatusFinal {
		t.Fatal("proof response should raise confidence to FINAL")
	}
	if updated.Confidence <= initial.Confidence {
		t.Fatal("proof response should increase confidence")
	}
}

func TestReplayMessageDropped(t *testing.T) {
	n := newTestNode("n", 100)
	peer := newTestNode("peer-a", 100)
	n.AddPeer(peer)
	inv := protocol.NewINVMessage(100, 30, sampleTransferID(7))
	if err := n.OnMessage("peer-a", inv); err != nil {
		t.Fatalf("first OnMessage(INV): %v", err)
	}
	firstSent := len(n.SentMessages())

	if err := n.OnMessage("peer-a", inv); err != nil {
		t.Fatalf("second OnMessage(INV): %v", err)
	}
	if len(n.SentMessages()) != firstSent {
		t.Fatal("replayed message should be dropped without new outbound messages")
	}
}

func TestLocalInputReuseRejected(t *testing.T) {
	n := newTestNode("n", 100)
	input, priv := seedKnownInput(n, nil, 9, 20)
	first := signedTransfer(input, priv, 15, 40)
	second := signedTransfer(input, priv, 20, 50)

	if err := n.AcceptLocalTransfer(first); err != nil {
		t.Fatalf("AcceptLocalTransfer(first): %v", err)
	}
	if err := n.AcceptLocalTransfer(second); !errors.Is(err, protocol.ErrInputConsumed) {
		t.Fatalf("expected ErrInputConsumed, got %v", err)
	}
}

func TestRemoteTXWithValidSignaturePasses(t *testing.T) {
	n := newTestNode("n", 100)
	input, priv := seedKnownInput(n, nil, 13, 25)
	tx := signedTransfer(input, priv, 25, 60)
	if err := n.OnMessage("peer-a", protocol.NewTXMessage(100, 30, tx)); err != nil {
		t.Fatalf("OnMessage(TX): %v", err)
	}
	if !n.hasTransfer(web4crypto.ComputeTransferID(tx)) {
		t.Fatal("valid signed remote TX was not stored")
	}
}

func TestRemoteTXTamperedPayloadFailsVerification(t *testing.T) {
	n := newTestNode("n", 100)
	input, priv := seedKnownInput(n, nil, 14, 25)
	tx := signedTransfer(input, priv, 25, 70)
	msg := protocol.NewTXMessage(100, 30, tx)
	payload, err := protocol.DecodeTXPayload(msg.Payload)
	if err != nil {
		t.Fatalf("DecodeTXPayload: %v", err)
	}
	payload.TransferBytes[len(payload.TransferBytes)-1] ^= 0xff
	msg.Payload = protocol.EncodeTXPayload(payload.TransferBytes, payload.Signature)

	if err := n.OnMessage("peer-a", msg); err == nil {
		t.Fatal("tampered TX payload should fail signature verification")
	}
}

func TestRemoteTXTamperedSignatureFailsVerification(t *testing.T) {
	n := newTestNode("n", 100)
	input, priv := seedKnownInput(n, nil, 15, 25)
	tx := signedTransfer(input, priv, 25, 80)
	msg := protocol.NewTXMessage(100, 30, tx)
	payload, err := protocol.DecodeTXPayload(msg.Payload)
	if err != nil {
		t.Fatalf("DecodeTXPayload: %v", err)
	}
	payload.Signature[0] ^= 0xff
	msg.Payload = protocol.EncodeTXPayload(payload.TransferBytes, payload.Signature)

	if err := n.OnMessage("peer-a", msg); err == nil {
		t.Fatal("tampered TX signature should fail verification")
	}
}

func newTestNode(id PeerID, now int64) *Node {
	n := NewNode(id)
	n.now = func() int64 { return now }
	n.replayCache = protocol.NewReplayCache(60)
	return n
}

func mustReceiveTX(t *testing.T, n *Node, from PeerID, tx web4crypto.Transfer) web4crypto.Hash {
	t.Helper()
	msg := protocol.NewTXMessage(n.now(), 30, tx)
	if err := n.OnMessage(from, msg); err != nil {
		t.Fatalf("OnMessage(TX): %v", err)
	}
	return web4crypto.ComputeTransferID(tx)
}

func seedKnownInput(primary *Node, mirror *Node, seed byte, amount uint64) (web4crypto.Value, ed25519.PrivateKey) {
	pub, priv := ownerKeys(seed)
	input := web4crypto.Value{
		Amount: amount,
		Unit:   "WEB4",
		Owner:  pub,
		Expiry: 1_800_000_000,
		Depth:  0,
	}
	input.ID = web4crypto.ComputeValueID(input)
	primary.valueStore[input.ID] = input
	if mirror != nil {
		mirror.valueStore[input.ID] = input
	}
	return input, priv
}

func signedTransfer(input web4crypto.Value, priv ed25519.PrivateKey, amount uint64, recipientSeed byte) web4crypto.Transfer {
	output := web4crypto.Value{
		Amount: amount,
		Unit:   input.Unit,
		Owner:  bytes.Repeat([]byte{recipientSeed}, ed25519.PublicKeySize),
		Expiry: 1_800_000_000,
		Depth:  input.Depth + 1,
	}
	output.ID = web4crypto.ComputeValueID(output)
	transfer := web4crypto.Transfer{
		Inputs:    []web4crypto.Hash{input.ID},
		Outputs:   []web4crypto.Value{output},
		Timestamp: 1_700_000_000,
	}
	transfer.Sig = web4crypto.SignCanonicalTransfer(priv, transfer)
	return transfer
}

func sampleTransferID(seed byte) web4crypto.Hash {
	var hash web4crypto.Hash
	for i := range hash {
		hash[i] = seed + byte(i)
	}
	return hash
}

func ownerKeys(seed byte) (ed25519.PublicKey, ed25519.PrivateKey) {
	seedBytes := bytes.Repeat([]byte{seed}, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seedBytes)
	pub := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pub, priv[32:])
	return pub, priv
}

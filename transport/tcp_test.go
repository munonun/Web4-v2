package transport

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	web4crypto "web4/crypto"
	"web4/node"
	"web4/protocol"
)

func TestMessageFrameRoundTrip(t *testing.T) {
	msg := protocol.NewINVMessage(1_700_000_100, 30, sampleTransferID(1))
	decoded, err := DecodeMessageFrame(bytes.NewReader(EncodeMessageFrame(msg)))
	if err != nil {
		t.Fatalf("DecodeMessageFrame: %v", err)
	}
	if !equalMessage(decoded, msg) {
		t.Fatal("decoded message mismatch")
	}
}

func TestMessageFrameRejectsShortFrame(t *testing.T) {
	frame := EncodeMessageFrame(protocol.NewINVMessage(1_700_000_100, 30, sampleTransferID(2)))
	_, err := DecodeMessageFrame(bytes.NewReader(frame[:len(frame)-1]))
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("expected short frame error, got %v", err)
	}
}

func TestMessageFrameHandlesPartialReads(t *testing.T) {
	msg := protocol.NewGETMessage(1_700_000_101, 30, sampleTransferID(3))
	decoded, err := DecodeMessageFrame(&chunkReader{data: EncodeMessageFrame(msg), size: 1})
	if err != nil {
		t.Fatalf("DecodeMessageFrame: %v", err)
	}
	if !equalMessage(decoded, msg) {
		t.Fatal("decoded message mismatch after partial reads")
	}
}

func TestTCPTransportSendsINV(t *testing.T) {
	received := make(chan struct {
		from string
		msg  protocol.Message
	}, 1)

	receiver := NewTCPTransport("127.0.0.1:0")
	if err := receiver.Start(func(from string, msg protocol.Message) error {
		received <- struct {
			from string
			msg  protocol.Message
		}{from: from, msg: msg}
		return nil
	}); err != nil {
		t.Fatalf("receiver.Start: %v", err)
	}
	t.Cleanup(func() { _ = receiver.Close() })

	sender := NewTCPTransport("127.0.0.1:0")
	if err := sender.Start(func(string, protocol.Message) error { return nil }); err != nil {
		t.Fatalf("sender.Start: %v", err)
	}
	t.Cleanup(func() { _ = sender.Close() })

	msg := protocol.NewINVMessage(1_700_000_100, 30, sampleTransferID(4))
	if err := sender.Send(receiver.Addr(), msg); err != nil {
		t.Fatalf("sender.Send: %v", err)
	}

	got := waitForValue(t, received)
	if got.from != sender.Addr() {
		t.Fatalf("unexpected sender identity: got %s want %s", got.from, sender.Addr())
	}
	if !equalMessage(got.msg, msg) {
		t.Fatal("received INV mismatch")
	}
}

func TestTCPTransportINVGETTXFlow(t *testing.T) {
	ta := NewTCPTransport("127.0.0.1:0")
	tb := NewTCPTransport("127.0.0.1:0")
	t.Cleanup(func() { _ = ta.Close() })
	t.Cleanup(func() { _ = tb.Close() })

	a := node.NewNode("A")
	b := node.NewNode("B")
	a.SetNowFunc(func() int64 { return 100 })
	b.SetNowFunc(func() int64 { return 100 })

	if err := AttachNode(a, ta); err != nil {
		t.Fatalf("AttachNode(A): %v", err)
	}
	if err := AttachNode(b, tb); err != nil {
		t.Fatalf("AttachNode(B): %v", err)
	}

	a.AddPeerID(node.PeerID(tb.Addr()))
	input, priv := seedKnownInput(a, b, 1, 25)
	tx := signedTransfer(input, priv, 25, 9)
	txID := web4crypto.ComputeTransferID(tx)

	if err := a.AcceptLocalTransfer(tx); err != nil {
		t.Fatalf("AcceptLocalTransfer: %v", err)
	}

	waitForCondition(t, func() bool { return b.HasTransfer(txID) })

	if !hasSentType(a, node.PeerID(tb.Addr()), protocol.INV) {
		t.Fatal("node A did not send INV over TCP")
	}
	if !hasSentType(b, node.PeerID(ta.Addr()), protocol.GET) {
		t.Fatal("node B did not send GET over TCP")
	}
	if !hasSentType(a, node.PeerID(tb.Addr()), protocol.TX) {
		t.Fatal("node A did not send TX over TCP")
	}
	if !b.HasTransfer(txID) {
		t.Fatal("node B did not store transfer")
	}
}

func TestTCPTransportMalformedPayloadDoesNotCrashListener(t *testing.T) {
	received := make(chan protocol.Message, 1)
	listener := NewTCPTransport("127.0.0.1:0")
	if err := listener.Start(func(_ string, msg protocol.Message) error {
		received <- msg
		return nil
	}); err != nil {
		t.Fatalf("listener.Start: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	rawConn, err := net.Dial("tcp", listener.Addr())
	if err != nil {
		t.Fatalf("net.Dial: %v", err)
	}
	if err := writeAll(rawConn, encodeHandshake("127.0.0.1:9999")); err != nil {
		t.Fatalf("write handshake: %v", err)
	}
	if err := writeAll(rawConn, []byte{0, 0, 0, 6, 1, 2, 3}); err != nil {
		t.Fatalf("write malformed frame: %v", err)
	}
	_ = rawConn.Close()

	sender := NewTCPTransport("127.0.0.1:0")
	if err := sender.Start(func(string, protocol.Message) error { return nil }); err != nil {
		t.Fatalf("sender.Start: %v", err)
	}
	t.Cleanup(func() { _ = sender.Close() })

	msg := protocol.NewINVMessage(1_700_000_100, 30, sampleTransferID(5))
	if err := sender.Send(listener.Addr(), msg); err != nil {
		t.Fatalf("sender.Send: %v", err)
	}

	got := waitForValue(t, received)
	if !equalMessage(got, msg) {
		t.Fatal("listener did not recover after malformed payload")
	}
}

func TestTCPTransportCloseShutsDownCleanly(t *testing.T) {
	first := NewTCPTransport("127.0.0.1:0")
	second := NewTCPTransport("127.0.0.1:0")
	if err := first.Start(func(string, protocol.Message) error { return nil }); err != nil {
		t.Fatalf("first.Start: %v", err)
	}
	if err := second.Start(func(string, protocol.Message) error { return nil }); err != nil {
		t.Fatalf("second.Start: %v", err)
	}

	msg := protocol.NewINVMessage(1_700_000_100, 30, sampleTransferID(6))
	if err := first.Send(second.Addr(), msg); err != nil {
		t.Fatalf("first.Send: %v", err)
	}

	closeDone := make(chan error, 2)
	go func() { closeDone <- first.Close() }()
	go func() { closeDone <- second.Close() }()

	for range 2 {
		select {
		case err := <-closeDone:
			if err != nil {
				t.Fatalf("Close: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Close timed out")
		}
	}

	if err := first.Send(second.Addr(), msg); err == nil {
		t.Fatal("Send should fail after Close")
	}
}

type chunkReader struct {
	data []byte
	size int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := r.size
	if n > len(r.data) {
		n = len(r.data)
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func waitForCondition(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met before timeout")
}

func waitForValue[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for value")
		var zero T
		return zero
	}
}

func hasSentType(n *node.Node, to node.PeerID, typ protocol.MsgType) bool {
	for _, sent := range n.SentMessages() {
		if sent.To == to && sent.Message.Type == typ {
			return true
		}
	}
	return false
}

func equalMessage(a, b protocol.Message) bool {
	return a.MessageID == b.MessageID &&
		a.Timestamp == b.Timestamp &&
		a.TTL == b.TTL &&
		a.Type == b.Type &&
		bytes.Equal(a.Payload, b.Payload)
}

func seedKnownInput(primary *node.Node, mirror *node.Node, seed byte, amount uint64) (web4crypto.Value, ed25519.PrivateKey) {
	pub, priv := ownerKeys(seed)
	input := web4crypto.Value{
		Amount: amount,
		Unit:   "WEB4",
		Owner:  pub,
		Expiry: 1_800_000_000,
		Depth:  0,
	}
	input.ID = web4crypto.ComputeValueID(input)
	primary.SeedValue(input)
	if mirror != nil {
		mirror.SeedValue(input)
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

func writeAll(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

package cli

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"time"

	web4crypto "web4/crypto"
	"web4/node"
	"web4/protocol"
	"web4/transport"
)

func runDemoCommand(args []string, w io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: web4 demo [basic|conflict|proof|tcp-basic]")
	}

	switch args[0] {
	case "basic":
		return RunDemoBasic(w)
	case "conflict":
		return RunDemoConflict(w)
	case "proof":
		return RunDemoProof(w)
	case "tcp-basic":
		return RunDemoTCPBasic(w)
	default:
		return fmt.Errorf("unknown demo: %s", args[0])
	}
}

func RunDemoTCPBasic(w io.Writer) error {
	return runDemoTCPBasic(w, []string{
		"127.0.0.1:9101",
		"127.0.0.1:9102",
		"127.0.0.1:9103",
	})
}

func RunDemoBasic(w io.Writer) error {
	a, b, c := basicNetwork(100)
	input, priv := seedKnownInput([]*node.Node{a, b, c}, 1, 25)
	tx := signedTransfer(input, priv, 25, 11)
	txID := web4crypto.ComputeTransferID(tx)

	if err := a.AcceptLocalTransfer(tx); err != nil {
		return err
	}

	for _, peer := range []struct {
		id   node.PeerID
		node *node.Node
	}{{"B", b}, {"C", c}} {
		if _, ok := findSentMessage(a, peer.id, protocol.INV); !ok {
			return fmt.Errorf("missing INV from A to %s", peer.id)
		}
		if _, ok := findSentMessage(peer.node, "A", protocol.GET); !ok {
			return fmt.Errorf("missing GET from %s to A", peer.id)
		}
		if _, ok := findSentMessage(a, peer.id, protocol.TX); !ok {
			return fmt.Errorf("missing TX from A to %s", peer.id)
		}
	}

	fprintf(w, "[A] created tx: %s\n", shortHash(txID))
	for _, peer := range []node.PeerID{"B", "C"} {
		fprintf(w, "[A -> %s] INV %s\n", peer, shortHash(txID))
		fprintf(w, "[%s -> A] GET %s\n", peer, shortHash(txID))
		fprintf(w, "[A -> %s] TX %s\n", peer, shortHash(txID))
		fprintf(w, "[%s] accepted tx %s\n", peer, shortHash(txID))
	}

	fprintf(w, "Final selected state:\n")
	for _, current := range []*node.Node{a, b, c} {
		selectedID, ok := current.SelectedForInput(input.ID)
		if !ok {
			return fmt.Errorf("missing selected lineage on node %s", current.ID)
		}
		state, ok := current.FinalityForTransfer(selectedID)
		if !ok {
			return fmt.Errorf("missing finality state on node %s", current.ID)
		}
		fprintf(w, "  Node %s -> %s (%s)\n", current.ID, shortHash(selectedID), state.Status)
	}
	return nil
}

func RunDemoConflict(w io.Writer) error {
	a, b, c := basicNetwork(100)
	b.AddPeer(c)
	c.AddPeer(b)
	input, priv := seedKnownInput([]*node.Node{a, b, c}, 2, 50)
	tx1 := signedTransfer(input, priv, 25, 21)
	tx2 := signedTransfer(input, priv, 40, 22)
	tx1ID := web4crypto.ComputeTransferID(tx1)
	tx2ID := web4crypto.ComputeTransferID(tx2)

	if err := a.AcceptLocalTransfer(tx1); err != nil {
		return err
	}
	if err := b.AcceptLocalTransfer(tx2); err != nil {
		return err
	}

	conflictSet := a.ConflictSet(input.ID)
	sortHashes(conflictSet)
	fprintf(w, "Conflict detected on input %s:\n", shortHash(input.ID))
	for _, txID := range conflictSet {
		fprintf(w, "  tx: %s\n", shortHash(txID))
	}

	fprintf(w, "\nScores:\n")
	for _, current := range []*node.Node{a, b, c} {
		fprintf(w, "  Node %s:\n", current.ID)
		for _, txID := range []web4crypto.Hash{tx1ID, tx2ID} {
			snapshot, ok := current.SelectionForInputTx(input.ID, txID)
			if !ok {
				return fmt.Errorf("missing selection snapshot on node %s", current.ID)
			}
			fprintf(w, "    %s = %.6f\n", shortHash(txID), snapshot.Score)
		}
	}

	fprintf(w, "\nSelected:\n")
	for _, current := range []*node.Node{a, b, c} {
		selectedID, ok := current.SelectedForInput(input.ID)
		if !ok {
			return fmt.Errorf("missing selected lineage on node %s", current.ID)
		}
		fprintf(w, "  Node %s -> %s\n", current.ID, shortHash(selectedID))
	}
	return nil
}

func RunDemoProof(w io.Writer) error {
	a, b, c := basicNetwork(100)
	b.AddPeer(c)
	c.AddPeer(b)
	input, priv := seedKnownInput([]*node.Node{a, b, c}, 3, 100_000)
	tx := signedTransfer(input, priv, 100_000, 31)
	txID := web4crypto.ComputeTransferID(tx)

	if err := b.AcceptLocalTransfer(tx); err != nil {
		return err
	}

	a.SetNowFunc(func() int64 { return 140 })
	b.SetNowFunc(func() int64 { return 140 })
	c.SetNowFunc(func() int64 { return 140 })

	before, ok := a.FinalityForTransfer(txID)
	if !ok {
		return fmt.Errorf("missing pre-proof finality state")
	}
	proofBefore := a.ProofWeightForTransfer(txID)

	reqToB := protocol.NewProofRequestMessage(140, 30, txID)
	reqToC := protocol.NewProofRequestMessage(140, 30, txID)
	if err := b.OnMessage("A", reqToB); err != nil {
		return err
	}
	if err := c.OnMessage("A", reqToC); err != nil {
		return err
	}

	responses := []struct {
		peer    *node.Node
		peerID  node.PeerID
		payload protocol.ProofResponsePayload
	}{
		{peer: b, peerID: "B"},
		{peer: c, peerID: "C"},
	}
	for i := range responses {
		msg, ok := findSentMessage(responses[i].peer, "A", protocol.PROOF_RESP)
		if !ok {
			return fmt.Errorf("missing PROOF_RESP from %s", responses[i].peerID)
		}
		payload, err := protocol.DecodeProofResponsePayload(msg.Payload)
		if err != nil {
			return err
		}
		responses[i].payload = payload
	}

	after, ok := a.FinalityForTransfer(txID)
	if !ok {
		return fmt.Errorf("missing post-proof finality state")
	}
	proofAfter := a.ProofWeightForTransfer(txID)

	fprintf(w, "Proof responses:\n")
	for _, response := range responses {
		fprintf(w, "  Peer %s -> seen=%t conflict=%t distance=%d seen_time=%d\n",
			response.peerID,
			response.payload.Seen,
			response.payload.Conflict,
			response.payload.Distance,
			response.payload.SeenTime,
		)
	}
	fprintf(w, "\nProof weight:\n")
	fprintf(w, "  before = %.2f\n", proofBefore)
	fprintf(w, "  after  = %.2f\n", proofAfter)
	fprintf(w, "\nConfidence:\n")
	fprintf(w, "  before = %.2f\n", before.Confidence)
	fprintf(w, "  after  = %.2f\n", after.Confidence)
	fprintf(w, "\nFinality: %s\n", after.Status)
	return nil
}

func runDemoTCPBasic(w io.Writer, listenAddrs []string) (err error) {
	if len(listenAddrs) != 3 {
		return fmt.Errorf("tcp-basic requires 3 listen addresses")
	}

	nodes := []*tcpDemoNode{
		{label: "A", node: node.NewNode("A"), tr: transport.NewTCPTransport(listenAddrs[0])},
		{label: "B", node: node.NewNode("B"), tr: transport.NewTCPTransport(listenAddrs[1])},
		{label: "C", node: node.NewNode("C"), tr: transport.NewTCPTransport(listenAddrs[2])},
	}
	for _, current := range nodes {
		current.node.SetNowFunc(func() int64 { return 100 })
	}
	defer func() {
		if closeErr := closeTCPDemoNodes(nodes); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	for _, current := range nodes {
		if err := transport.AttachNode(current.node, current.tr); err != nil {
			return err
		}
		current.addr = current.tr.Addr()
	}

	a, b, c := nodes[0], nodes[1], nodes[2]
	a.node.AddPeerID(node.PeerID(b.addr))
	a.node.AddPeerID(node.PeerID(c.addr))
	b.node.AddPeerID(node.PeerID(a.addr))
	c.node.AddPeerID(node.PeerID(a.addr))

	input, priv := seedKnownInput([]*node.Node{a.node, b.node, c.node}, 1, 25)
	tx := signedTransfer(input, priv, 25, 11)
	txID := web4crypto.ComputeTransferID(tx)

	if err := a.node.AcceptLocalTransfer(tx); err != nil {
		return err
	}
	if err := waitForTCPTransfer(b.node, txID); err != nil {
		return err
	}
	if err := waitForTCPTransfer(c.node, txID); err != nil {
		return err
	}

	for _, peer := range []struct {
		label string
		node  *node.Node
		addr  string
	}{{label: "B", node: b.node, addr: b.addr}, {label: "C", node: c.node, addr: c.addr}} {
		if _, ok := findSentMessage(a.node, node.PeerID(peer.addr), protocol.INV); !ok {
			return fmt.Errorf("missing INV from A to %s", peer.label)
		}
		if _, ok := findSentMessage(peer.node, node.PeerID(a.addr), protocol.GET); !ok {
			return fmt.Errorf("missing GET from %s to A", peer.label)
		}
		if _, ok := findSentMessage(a.node, node.PeerID(peer.addr), protocol.TX); !ok {
			return fmt.Errorf("missing TX from A to %s", peer.label)
		}
	}

	fprintf(w, "[TCP BASIC]\n")
	for _, current := range nodes {
		fprintf(w, "[%s] listening on %s\n", current.label, current.addr)
	}
	fprintf(w, "\n")

	fprintf(w, "[A] created tx: %s\n", shortHash(txID))
	for _, peer := range []string{"B", "C"} {
		fprintf(w, "[A -> %s] INV %s\n", peer, shortHash(txID))
		fprintf(w, "[%s -> A] GET %s\n", peer, shortHash(txID))
		fprintf(w, "[A -> %s] TX %s\n", peer, shortHash(txID))
		fprintf(w, "[%s] accepted tx %s\n", peer, shortHash(txID))
	}
	fprintf(w, "\nFinal selected state:\n")
	for _, current := range nodes {
		selectedID, ok := current.node.SelectedForInput(input.ID)
		if !ok {
			return fmt.Errorf("missing selected lineage on node %s", current.label)
		}
		state, ok := current.node.FinalityForTransfer(selectedID)
		if !ok {
			return fmt.Errorf("missing finality state on node %s", current.label)
		}
		fprintf(w, "  Node %s -> %s (%s)\n", current.label, shortHash(selectedID), state.Status)
	}
	return nil
}

func basicNetwork(now int64) (*node.Node, *node.Node, *node.Node) {
	a := node.NewNode("A")
	b := node.NewNode("B")
	c := node.NewNode("C")
	for _, current := range []*node.Node{a, b, c} {
		current.SetNowFunc(func() int64 { return int64(now) })
	}
	a.AddPeer(b)
	a.AddPeer(c)
	b.AddPeer(a)
	c.AddPeer(a)
	return a, b, c
}

func seedKnownInput(nodes []*node.Node, seed byte, amount uint64) (web4crypto.Value, ed25519.PrivateKey) {
	pub, priv := ownerKeys(seed)
	input := web4crypto.Value{
		Amount: amount,
		Unit:   "WEB4",
		Owner:  pub,
		Expiry: 1_800_000_000,
		Depth:  0,
	}
	input.ID = web4crypto.ComputeValueID(input)
	for _, current := range nodes {
		current.SeedValue(input)
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

func ownerKeys(seed byte) (ed25519.PublicKey, ed25519.PrivateKey) {
	seedBytes := bytes.Repeat([]byte{seed}, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seedBytes)
	pub := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pub, priv[32:])
	return pub, priv
}

func closeTCPDemoNodes(nodes []*tcpDemoNode) error {
	var firstErr error
	for i := len(nodes) - 1; i >= 0; i-- {
		if err := nodes[i].tr.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func findSentMessage(n *node.Node, to node.PeerID, msgType protocol.MsgType) (protocol.Message, bool) {
	for _, sent := range n.SentMessages() {
		if sent.To == to && sent.Message.Type == msgType {
			return sent.Message, true
		}
	}
	return protocol.Message{}, false
}

func shortHash(hash web4crypto.Hash) string {
	return hex.EncodeToString(hash[:6])
}

func sortHashes(values []web4crypto.Hash) {
	sort.Slice(values, func(i, j int) bool {
		return bytes.Compare(values[i][:], values[j][:]) < 0
	})
}

func fprintf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

type tcpDemoNode struct {
	label string
	node  *node.Node
	tr    *transport.TCPTransport
	addr  string
}

func waitForTCPTransfer(n *node.Node, txID web4crypto.Hash) error {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if n.HasTransfer(txID) {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for transfer %s", shortHash(txID))
}

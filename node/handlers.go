package node

import (
	"bytes"
	"crypto/ed25519"
	"fmt"

	web4crypto "web4/crypto"
	"web4/protocol"
)

func (n *Node) handleINV(from PeerID, msg protocol.Message) error {
	txID, err := protocol.DecodeINVID(msg.Payload)
	if err != nil {
		return err
	}
	if n.hasTransfer(txID) {
		return nil
	}
	return n.send(from, protocol.NewGETMessage(n.now(), msg.TTL, txID))
}

func (n *Node) handleGET(from PeerID, msg protocol.Message) error {
	txID, err := protocol.DecodeGETID(msg.Payload)
	if err != nil {
		return err
	}
	record, ok := n.txStore[txID]
	if !ok {
		return nil
	}
	return n.send(from, protocol.NewTXMessage(n.now(), msg.TTL, record.Tx))
}

func (n *Node) handleTX(from PeerID, msg protocol.Message) error {
	payload, err := protocol.DecodeTXPayload(msg.Payload)
	if err != nil {
		return err
	}
	tx, err := web4crypto.DecodeTransfer(payload.TransferBytes)
	if err != nil {
		return err
	}
	tx.Sig = payload.Signature
	tx = n.normalizeTransfer(tx)
	if err := n.validateTransfer(tx); err != nil {
		return err
	}

	txID, inserted, err := n.upsertTransfer(tx, payload.TransferBytes, from)
	if err != nil {
		return err
	}
	n.recomputeSelectionForTransfer(txID)
	if !inserted {
		return nil
	}
	for _, peerID := range n.peerIDsExcept(from) {
		if err := n.send(peerID, protocol.NewINVMessage(n.now(), msg.TTL, txID)); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) handleProofRequest(from PeerID, msg protocol.Message) error {
	txID, err := protocol.DecodeProofRequestPayload(msg.Payload)
	if err != nil {
		return err
	}
	payload := protocol.ProofResponsePayload{TargetID: txID, Distance: 1}
	if record, ok := n.txStore[txID]; ok {
		payload.Seen = true
		payload.SeenTime = record.FirstSeen
		payload.Conflict = n.hasConflict(txID)
	}
	return n.send(from, protocol.NewProofResponseMessage(n.now(), msg.TTL, payload))
}

func (n *Node) handleProofResponse(from PeerID, msg protocol.Message) error {
	payload, err := protocol.DecodeProofResponsePayload(msg.Payload)
	if err != nil {
		return err
	}
	n.updateProof(payload.TargetID, ProofRecord{
		PeerID:   from,
		Seen:     payload.Seen,
		SeenTime: payload.SeenTime,
		Conflict: payload.Conflict,
		Distance: payload.Distance,
	})
	if n.hasTransfer(payload.TargetID) {
		n.recomputeSelectionForTransfer(payload.TargetID)
	}
	return nil
}

func (n *Node) validateTransfer(tx web4crypto.Transfer) error {
	if len(tx.Inputs) == 0 {
		return fmt.Errorf("transfer has no inputs")
	}
	if len(tx.Outputs) == 0 {
		return fmt.Errorf("transfer has no outputs")
	}
	seenInputs := make(map[web4crypto.Hash]struct{}, len(tx.Inputs))
	for _, input := range tx.Inputs {
		if _, ok := seenInputs[input]; ok {
			return protocol.ErrInputConsumed
		}
		seenInputs[input] = struct{}{}
	}
	for i := range tx.Outputs {
		output := tx.Outputs[i]
		if output.Expiry != 0 && output.Expiry <= n.now() {
			return fmt.Errorf("output %d expired", i)
		}
	}
	owner, err := n.resolveSigner(tx.Inputs)
	if err != nil {
		return err
	}
	if len(owner) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid input owner public key length: %d", len(owner))
	}
	if !web4crypto.VerifyCanonicalTransfer(ed25519.PublicKey(owner), tx, tx.Sig) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func (n *Node) hasConflict(txID web4crypto.Hash) bool {
	record, ok := n.txStore[txID]
	if !ok {
		return false
	}
	for _, input := range record.Tx.Inputs {
		if len(n.conflictSets[input]) > 1 {
			return true
		}
	}
	return false
}

func (n *Node) normalizeTransfer(tx web4crypto.Transfer) web4crypto.Transfer {
	normalized := tx
	normalized.Outputs = append([]web4crypto.Value(nil), tx.Outputs...)
	for i, output := range normalized.Outputs {
		output.ID = web4crypto.ComputeValueID(output)
		normalized.Outputs[i] = output
	}
	normalized.ID = web4crypto.ComputeTransferID(normalized)
	return normalized
}

func (n *Node) resolveSigner(inputs []web4crypto.Hash) ([]byte, error) {
	var owner []byte
	for _, input := range inputs {
		value, ok := n.valueStore[input]
		if !ok {
			return nil, fmt.Errorf("unknown input: %x", input)
		}
		if owner == nil {
			owner = append([]byte(nil), value.Owner...)
			continue
		}
		if !bytes.Equal(owner, value.Owner) {
			return nil, fmt.Errorf("inputs have different owners")
		}
	}
	return owner, nil
}

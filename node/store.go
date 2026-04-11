package node

import (
	"bytes"

	web4crypto "web4/crypto"
)

func (n *Node) hasTransfer(txID web4crypto.Hash) bool {
	_, ok := n.txStore[txID]
	return ok
}

func (n *Node) upsertTransfer(tx web4crypto.Transfer, canonical []byte, from PeerID) (web4crypto.Hash, bool, error) {
	tx = n.normalizeTransfer(tx)
	tx.ID = web4crypto.ComputeTransferID(tx)
	if existing, ok := n.txStore[tx.ID]; ok {
		n.observePeer(tx.ID, from)
		existing.SeenCount = len(existing.Peers)
		return tx.ID, false, nil
	}

	n.arrivalSeq++
	record := &TxRecord{
		Tx:         tx,
		Canonical:  bytes.Clone(canonical),
		FirstSeen:  n.now(),
		ArrivalSeq: n.arrivalSeq,
		Peers:      make(map[PeerID]int64),
	}
	n.txStore[tx.ID] = record
	for _, output := range tx.Outputs {
		n.valueStore[output.ID] = output
	}
	n.observePeer(tx.ID, from)
	record.SeenCount = len(record.Peers)
	n.addConflictMembership(tx.ID, tx.Inputs)
	return tx.ID, true, nil
}

func (n *Node) observePeer(txID web4crypto.Hash, from PeerID) {
	if from == "" {
		return
	}
	peers, ok := n.peerObservations[txID]
	if !ok {
		peers = make(map[PeerID]int64)
		n.peerObservations[txID] = peers
	}
	peers[from] = n.now()
	if record, ok := n.txStore[txID]; ok {
		record.Peers[from] = peers[from]
		record.SeenCount = len(record.Peers)
	}
}

func (n *Node) addConflictMembership(txID web4crypto.Hash, inputs []web4crypto.Hash) {
	for _, input := range inputs {
		txIDs := n.conflictSets[input]
		if containsHash(txIDs, txID) {
			continue
		}
		n.conflictSets[input] = append(txIDs, txID)
	}
}

func (n *Node) updateProof(txID web4crypto.Hash, proof ProofRecord) {
	proofs, ok := n.proofStore[txID]
	if !ok {
		proofs = make(map[PeerID]ProofRecord)
		n.proofStore[txID] = proofs
	}
	proofs[proof.PeerID] = proof
}

func containsHash(values []web4crypto.Hash, target web4crypto.Hash) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

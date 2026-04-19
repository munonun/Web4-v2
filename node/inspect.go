package node

import (
	"bytes"
	"sort"

	web4crypto "web4/crypto"
)

type SelectedLineageEntry struct {
	Input     web4crypto.Hash
	TxID      web4crypto.Hash
	Score     float64
	UpdatedAt int64
}

type FinalityEntry struct {
	TxID   web4crypto.Hash
	State  FinalityState
	Amount uint64
}

type ConflictSetEntry struct {
	Input web4crypto.Hash
	TxIDs []web4crypto.Hash
}

func (n *Node) PeerCount() int {
	return len(n.peers)
}

func (n *Node) TransferCount() int {
	return len(n.txStore)
}

func (n *Node) ConflictSetCount() int {
	return len(n.conflictSets)
}

func (n *Node) SelectedLineageCount() int {
	return len(n.selectedLineage)
}

func (n *Node) Value(valueID web4crypto.Hash) (web4crypto.Value, bool) {
	value, ok := n.valueStore[valueID]
	return value, ok
}

func (n *Node) Values() []web4crypto.Value {
	values := make([]web4crypto.Value, 0, len(n.valueStore))
	for _, value := range n.valueStore {
		values = append(values, value)
	}
	sort.Slice(values, func(i, j int) bool {
		return bytes.Compare(values[i].ID[:], values[j].ID[:]) < 0
	})
	return values
}

func (n *Node) SelectedLineages() []SelectedLineageEntry {
	entries := make([]SelectedLineageEntry, 0, len(n.selectedLineage))
	for input, selected := range n.selectedLineage {
		entries = append(entries, SelectedLineageEntry{
			Input:     input,
			TxID:      selected.TxID,
			Score:     selected.Score,
			UpdatedAt: selected.UpdatedAt,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].Input[:], entries[j].Input[:]) < 0
	})
	return entries
}

func (n *Node) FinalityEntries() []FinalityEntry {
	entries := make([]FinalityEntry, 0, len(n.finalityState))
	for txID, state := range n.finalityState {
		entries = append(entries, FinalityEntry{
			TxID:   txID,
			State:  state,
			Amount: transferAmount(n.txStore[txID].Tx),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].TxID[:], entries[j].TxID[:]) < 0
	})
	return entries
}

func (n *Node) ConflictSetEntries() []ConflictSetEntry {
	entries := make([]ConflictSetEntry, 0, len(n.conflictSets))
	for input, txIDs := range n.conflictSets {
		cloned := append([]web4crypto.Hash(nil), txIDs...)
		sort.Slice(cloned, func(i, j int) bool {
			return bytes.Compare(cloned[i][:], cloned[j][:]) < 0
		})
		entries = append(entries, ConflictSetEntry{Input: input, TxIDs: cloned})
	}
	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].Input[:], entries[j].Input[:]) < 0
	})
	return entries
}

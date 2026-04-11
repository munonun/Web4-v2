package node

import (
	web4crypto "web4/crypto"
	"web4/protocol"
)

type PeerID string

type Status string

const (
	StatusPending  Status = "PENDING"
	StatusSelected Status = "SELECTED"
	StatusFinal    Status = "FINAL"
)

type TxRecord struct {
	Tx         web4crypto.Transfer
	Canonical  []byte
	FirstSeen  int64
	ArrivalSeq int64
	SeenCount  int
	Peers      map[PeerID]int64
}

type SelectedLineage struct {
	TxID      web4crypto.Hash
	Score     float64
	UpdatedAt int64
}

type FinalityState struct {
	Score      float64
	Confidence float64
	Status     Status
}

type SelectionSnapshot struct {
	TxID                web4crypto.Hash
	Score               float64
	FirstSeenScore      float64
	PropagationScore    float64
	TimeStabilityScore  float64
	NeighborhoodScore   float64
	ConflictCleanliness float64
}

type ProofRecord struct {
	PeerID   PeerID
	Seen     bool
	SeenTime int64
	Conflict bool
	Distance int
}

type SentMessage struct {
	To      PeerID
	Message protocol.Message
}

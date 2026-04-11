package node

import (
	"bytes"
	"math"
	"sort"

	web4crypto "web4/crypto"
)

const (
	weightFirstSeen   = 0.35
	weightPropagation = 0.20
	weightTime        = 0.20
	weightProof       = 0.20
	weightConflict    = 0.05

	proofSmoothing = 1.0
	stabilityTau   = 30.0
)

type scoreBreakdown struct {
	TxID                web4crypto.Hash
	Score               float64
	FirstSeenScore      float64
	PropagationScore    float64
	TimeStabilityScore  float64
	NeighborhoodScore   float64
	ConflictCleanliness float64
}

func (n *Node) SelectionForInputTx(input, txID web4crypto.Hash) (SelectionSnapshot, bool) {
	txIDs := n.conflictSets[input]
	if len(txIDs) == 0 || !containsHash(txIDs, txID) {
		return SelectionSnapshot{}, false
	}
	ranked := n.rankByArrival(txIDs)
	for rank, candidate := range ranked {
		if candidate != txID {
			continue
		}
		breakdown := n.scoreTx(input, ranked, rank)
		return SelectionSnapshot{
			TxID:                breakdown.TxID,
			Score:               breakdown.Score,
			FirstSeenScore:      breakdown.FirstSeenScore,
			PropagationScore:    breakdown.PropagationScore,
			TimeStabilityScore:  breakdown.TimeStabilityScore,
			NeighborhoodScore:   breakdown.NeighborhoodScore,
			ConflictCleanliness: breakdown.ConflictCleanliness,
		}, true
	}
	return SelectionSnapshot{}, false
}

func (n *Node) recomputeSelectionForTransfer(txID web4crypto.Hash) {
	record, ok := n.txStore[txID]
	if !ok {
		return
	}
	for _, input := range record.Tx.Inputs {
		n.recomputeSelectionForInput(input)
	}
	n.updateFinalityForTransfer(txID)
}

func (n *Node) recomputeSelectionForInput(input web4crypto.Hash) {
	txIDs := n.conflictSets[input]
	if len(txIDs) == 0 {
		delete(n.selectedLineage, input)
		return
	}

	best := n.selectBest(input, txIDs)
	n.selectedLineage[input] = SelectedLineage{
		TxID:      best.TxID,
		Score:     best.Score,
		UpdatedAt: n.now(),
	}

	for _, txID := range txIDs {
		n.updateFinalityForTransfer(txID)
	}
}

func (n *Node) selectBest(input web4crypto.Hash, txIDs []web4crypto.Hash) scoreBreakdown {
	ranked := n.rankByArrival(txIDs)
	best := n.scoreTx(input, ranked, 0)
	for rank := 1; rank < len(ranked); rank++ {
		candidate := n.scoreTx(input, ranked, rank)
		if betterScore(candidate, best) {
			best = candidate
		}
	}
	return best
}

func betterScore(left, right scoreBreakdown) bool {
	if left.Score != right.Score {
		return left.Score > right.Score
	}
	if left.FirstSeenScore != right.FirstSeenScore {
		return left.FirstSeenScore > right.FirstSeenScore
	}
	if left.PropagationScore != right.PropagationScore {
		return left.PropagationScore > right.PropagationScore
	}
	if left.TimeStabilityScore != right.TimeStabilityScore {
		return left.TimeStabilityScore > right.TimeStabilityScore
	}
	if left.NeighborhoodScore != right.NeighborhoodScore {
		return left.NeighborhoodScore > right.NeighborhoodScore
	}
	return bytes.Compare(left.TxID[:], right.TxID[:]) < 0
}

func (n *Node) scoreTx(input web4crypto.Hash, ranked []web4crypto.Hash, rank int) scoreBreakdown {
	txID := ranked[rank]
	firstSeen := 1.0 / (1.0 + float64(rank))
	propagation := math.Log(1.0 + float64(len(n.peerObservations[txID])))
	timeStability := n.timeStability(txID)
	proofWeight := n.proofWeight(txID)
	neighborhood := proofWeight / (proofWeight + proofSmoothing)
	cleanliness := n.conflictCleanliness(input, txID)
	score := weightFirstSeen*firstSeen + weightPropagation*propagation + weightTime*timeStability + weightProof*neighborhood + weightConflict*cleanliness

	return scoreBreakdown{
		TxID:                txID,
		Score:               score,
		FirstSeenScore:      firstSeen,
		PropagationScore:    propagation,
		TimeStabilityScore:  timeStability,
		NeighborhoodScore:   neighborhood,
		ConflictCleanliness: cleanliness,
	}
}

func (n *Node) rankByArrival(txIDs []web4crypto.Hash) []web4crypto.Hash {
	ranked := append([]web4crypto.Hash(nil), txIDs...)
	sort.Slice(ranked, func(i, j int) bool {
		left := n.txStore[ranked[i]]
		right := n.txStore[ranked[j]]
		if left.ArrivalSeq != right.ArrivalSeq {
			return left.ArrivalSeq < right.ArrivalSeq
		}
		return bytes.Compare(ranked[i][:], ranked[j][:]) < 0
	})
	return ranked
}

func (n *Node) timeStability(txID web4crypto.Hash) float64 {
	record, ok := n.txStore[txID]
	if !ok {
		return 0
	}
	age := float64(n.now() - record.FirstSeen)
	if age < 0 {
		age = 0
	}
	if age >= stabilityTau {
		return 1
	}
	return age / stabilityTau
}

func (n *Node) proofWeight(txID web4crypto.Hash) float64 {
	proofs := n.proofStore[txID]
	weight := 0.0
	for _, proof := range proofs {
		if !proof.Seen || proof.Conflict {
			continue
		}
		weight += distanceWeight(proof.Distance)
	}
	return weight
}

func (n *Node) conflictCleanliness(input, txID web4crypto.Hash) float64 {
	conflictSignals := 0.0
	maxSignals := float64(len(n.conflictSets[input]) - 1)
	if len(n.conflictSets[input]) > 1 {
		conflictSignals += float64(len(n.conflictSets[input]) - 1)
	}
	for _, proof := range n.proofStore[txID] {
		maxSignals++
		if proof.Conflict {
			conflictSignals++
		}
	}
	return 1.0 - (conflictSignals / (maxSignals + 1.0))
}

func distanceWeight(distance int) float64 {
	switch distance {
	case 0, 1:
		return 1.0
	case 2:
		return 0.6
	default:
		return 0.3
	}
}

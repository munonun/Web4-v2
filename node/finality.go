package node

import (
	"math"

	web4crypto "web4/crypto"
)

const (
	confidencePropagation = 0.30
	confidenceTime        = 0.35
	confidenceProof       = 0.35

	thresholdSmall  = 0.45
	thresholdMedium = 0.70
	thresholdLarge  = 0.88

	tierSmallMax  = 50_000
	tierMediumMax = 500_000
)

func (n *Node) updateFinalityForTransfer(txID web4crypto.Hash) {
	record, ok := n.txStore[txID]
	if !ok {
		return
	}

	selected := true
	var score float64
	for _, input := range record.Tx.Inputs {
		current, ok := n.selectedLineage[input]
		if !ok || current.TxID != txID {
			selected = false
			break
		}
		score = current.Score
	}

	state := FinalityState{Score: score, Status: StatusPending}
	if selected {
		confidence := n.computeConfidence(txID)
		state.Confidence = confidence
		state.Status = StatusSelected
		if confidence >= finalityThreshold(record.Tx) {
			state.Status = StatusFinal
		}
	}
	n.finalityState[txID] = state
}

func (n *Node) computeConfidence(txID web4crypto.Hash) float64 {
	propagation := propagationScore(n.peerObservations[txID])
	timeStability := n.timeStability(txID)
	proofWeight := n.proofWeight(txID)
	neighborhood := proofWeight / (proofWeight + proofSmoothing)
	return confidencePropagation*propagation + confidenceTime*timeStability + confidenceProof*neighborhood
}

func propagationScore(peers map[PeerID]int64) float64 {
	return weightLogCount(len(peers))
}

func weightLogCount(count int) float64 {
	if count <= 0 {
		return 0
	}
	return math.Log(1 + float64(count))
}

func finalityThreshold(tx web4crypto.Transfer) float64 {
	amount := transferAmount(tx)
	switch {
	case amount < tierSmallMax:
		return thresholdSmall
	case amount < tierMediumMax:
		return thresholdMedium
	default:
		return thresholdLarge
	}
}

func transferAmount(tx web4crypto.Transfer) uint64 {
	var total uint64
	for _, output := range tx.Outputs {
		total += output.Amount
	}
	return total
}

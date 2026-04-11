package node

import (
	"errors"
	"fmt"
	"sort"
	"time"

	web4crypto "web4/crypto"
	"web4/protocol"
)

var ErrUnknownPeer = errors.New("unknown peer")

type Node struct {
	ID PeerID

	now func() int64

	txStore          map[web4crypto.Hash]*TxRecord
	valueStore       map[web4crypto.Hash]web4crypto.Value
	conflictSets     map[web4crypto.Hash][]web4crypto.Hash
	selectedLineage  map[web4crypto.Hash]SelectedLineage
	finalityState    map[web4crypto.Hash]FinalityState
	peerObservations map[web4crypto.Hash]map[PeerID]int64
	proofStore       map[web4crypto.Hash]map[PeerID]ProofRecord
	replayCache      *protocol.ReplayCache
	inputTracker     *protocol.InputTracker
	peers            map[PeerID]*Node
	sent             []SentMessage
	arrivalSeq       int64
}

func NewNode(id PeerID) *Node {
	return &Node{
		ID:               id,
		now:              func() int64 { return time.Now().Unix() },
		txStore:          make(map[web4crypto.Hash]*TxRecord),
		valueStore:       make(map[web4crypto.Hash]web4crypto.Value),
		conflictSets:     make(map[web4crypto.Hash][]web4crypto.Hash),
		selectedLineage:  make(map[web4crypto.Hash]SelectedLineage),
		finalityState:    make(map[web4crypto.Hash]FinalityState),
		peerObservations: make(map[web4crypto.Hash]map[PeerID]int64),
		proofStore:       make(map[web4crypto.Hash]map[PeerID]ProofRecord),
		replayCache:      protocol.NewReplayCache(60),
		inputTracker:     protocol.NewInputTracker(),
		peers:            make(map[PeerID]*Node),
	}
}

func (n *Node) AddPeer(peer *Node) {
	n.peers[peer.ID] = peer
}

func (n *Node) SetNowFunc(now func() int64) {
	n.now = now
}

func (n *Node) SeedValue(value web4crypto.Value) {
	value.ID = web4crypto.ComputeValueID(value)
	n.valueStore[value.ID] = value
}

func (n *Node) HasTransfer(txID web4crypto.Hash) bool {
	return n.hasTransfer(txID)
}

func (n *Node) ConflictSet(input web4crypto.Hash) []web4crypto.Hash {
	values := append([]web4crypto.Hash(nil), n.conflictSets[input]...)
	return values
}

func (n *Node) SelectedForInput(input web4crypto.Hash) (web4crypto.Hash, bool) {
	selected, ok := n.selectedLineage[input]
	if !ok {
		return web4crypto.Hash{}, false
	}
	return selected.TxID, true
}

func (n *Node) FinalityForTransfer(txID web4crypto.Hash) (FinalityState, bool) {
	state, ok := n.finalityState[txID]
	return state, ok
}

func (n *Node) Transfer(txID web4crypto.Hash) (web4crypto.Transfer, bool) {
	record, ok := n.txStore[txID]
	if !ok {
		return web4crypto.Transfer{}, false
	}
	return record.Tx, true
}

func (n *Node) ProofWeightForTransfer(txID web4crypto.Hash) float64 {
	return n.proofWeight(txID)
}

func (n *Node) SentMessages() []SentMessage {
	out := make([]SentMessage, len(n.sent))
	copy(out, n.sent)
	return out
}

func (n *Node) OnMessage(from PeerID, msg protocol.Message) error {
	n.replayCache.PruneExpired(n.now())
	if n.replayCache.IsSeen(msg.MessageID) {
		return nil
	}
	n.replayCache.MarkSeen(msg.MessageID)

	switch msg.Type {
	case protocol.INV:
		return n.handleINV(from, msg)
	case protocol.GET:
		return n.handleGET(from, msg)
	case protocol.TX:
		return n.handleTX(from, msg)
	case protocol.PROOF_REQ:
		return n.handleProofRequest(from, msg)
	case protocol.PROOF_RESP:
		return n.handleProofResponse(from, msg)
	case protocol.ACK, protocol.ERROR:
		return nil
	default:
		return fmt.Errorf("unsupported message type: %d", msg.Type)
	}
}

func (n *Node) AcceptLocalTransfer(tx web4crypto.Transfer) error {
	tx = n.normalizeTransfer(tx)
	if err := n.validateTransfer(tx); err != nil {
		return err
	}
	if err := n.inputTracker.ConsumeInputs(tx.Inputs); err != nil {
		return err
	}
	canonical := web4crypto.EncodeTransfer(tx)
	tx.ID = web4crypto.ComputeTransferID(tx)
	txID, _, err := n.upsertTransfer(tx, canonical, "")
	if err != nil {
		return err
	}
	n.recomputeSelectionForTransfer(txID)
	for _, peerID := range n.peerIDsExcept("") {
		if err := n.send(peerID, protocol.NewINVMessage(n.now(), 30, tx.ID)); err != nil {
			return err
		}
	}
	return nil
}

func (n *Node) send(to PeerID, msg protocol.Message) error {
	peer, ok := n.peers[to]
	if !ok {
		return ErrUnknownPeer
	}
	n.sent = append(n.sent, SentMessage{To: to, Message: msg})
	return peer.OnMessage(n.ID, msg)
}

func (n *Node) peerIDsExcept(skip PeerID) []PeerID {
	peerIDs := make([]PeerID, 0, len(n.peers))
	for peerID := range n.peers {
		if peerID == skip {
			continue
		}
		peerIDs = append(peerIDs, peerID)
	}
	sort.Slice(peerIDs, func(i, j int) bool { return peerIDs[i] < peerIDs[j] })
	return peerIDs
}

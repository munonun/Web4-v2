package protocol

import (
	"errors"
	"testing"

	web4crypto "web4/crypto"
)

func TestReplayCacheDetectsDuplicateMessage(t *testing.T) {
	cache := newReplayCacheWithClock(30, func() int64 { return 100 })
	messageID := [16]byte{1, 2, 3}

	if cache.IsSeen(messageID) {
		t.Fatal("message should not be seen before marking")
	}

	cache.MarkSeen(messageID)
	if !cache.IsSeen(messageID) {
		t.Fatal("message should be seen after marking")
	}
}

func TestReplayCachePrunesExpiredEntries(t *testing.T) {
	cache := newReplayCacheWithClock(10, func() int64 { return 50 })
	messageID := [16]byte{9, 9, 9}

	cache.MarkSeen(messageID)
	if !cache.IsSeen(messageID) {
		t.Fatal("message should be seen before pruning")
	}

	cache.PruneExpired(60)
	if cache.IsSeen(messageID) {
		t.Fatal("message should be pruned after TTL expiry")
	}
}

func TestInputTrackerRejectsReuse(t *testing.T) {
	tracker := NewInputTracker()
	input := web4crypto.Hash{1, 2, 3}

	if err := tracker.MarkConsumed(input); err != nil {
		t.Fatalf("MarkConsumed: %v", err)
	}
	if err := tracker.MarkConsumed(input); !errors.Is(err, ErrInputConsumed) {
		t.Fatalf("expected ErrInputConsumed, got %v", err)
	}
}

func TestInputTrackerConsumeInputsRejectsBatchReuse(t *testing.T) {
	tracker := NewInputTracker()
	inputs := []web4crypto.Hash{{1}, {2}}
	if err := tracker.ConsumeInputs(inputs); err != nil {
		t.Fatalf("ConsumeInputs: %v", err)
	}

	if err := tracker.ConsumeInputs([]web4crypto.Hash{{2}, {3}}); !errors.Is(err, ErrInputConsumed) {
		t.Fatalf("expected ErrInputConsumed, got %v", err)
	}
	if tracker.IsConsumed(web4crypto.Hash{3}) {
		t.Fatal("failed batch must not partially mark new inputs")
	}
}

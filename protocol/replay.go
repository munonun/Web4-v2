package protocol

import (
	"errors"
	"time"

	web4crypto "web4/crypto"
)

var ErrInputConsumed = errors.New("input already consumed locally")

type ReplayCache struct {
	defaultTTL int64
	now        func() int64
	seen       map[[16]byte]int64
}

func NewReplayCache(defaultTTL int64) *ReplayCache {
	return &ReplayCache{
		defaultTTL: defaultTTL,
		now: func() int64 {
			return time.Now().Unix()
		},
		seen: make(map[[16]byte]int64),
	}
}

func (c *ReplayCache) MarkSeen(messageID [16]byte) {
	expiresAt := int64(0)
	if c.defaultTTL > 0 {
		expiresAt = c.now() + c.defaultTTL
	}
	c.seen[messageID] = expiresAt
}

func (c *ReplayCache) IsSeen(messageID [16]byte) bool {
	_, ok := c.seen[messageID]
	return ok
}

func (c *ReplayCache) PruneExpired(now int64) {
	for messageID, expiresAt := range c.seen {
		if expiresAt > 0 && expiresAt <= now {
			delete(c.seen, messageID)
		}
	}
}

type InputTracker struct {
	consumed map[web4crypto.Hash]struct{}
}

func NewInputTracker() *InputTracker {
	return &InputTracker{consumed: make(map[web4crypto.Hash]struct{})}
}

func (t *InputTracker) MarkConsumed(input web4crypto.Hash) error {
	if t.IsConsumed(input) {
		return ErrInputConsumed
	}
	t.consumed[input] = struct{}{}
	return nil
}

func (t *InputTracker) IsConsumed(input web4crypto.Hash) bool {
	_, ok := t.consumed[input]
	return ok
}

func (t *InputTracker) ConsumeInputs(inputs []web4crypto.Hash) error {
	seenInBatch := make(map[web4crypto.Hash]struct{}, len(inputs))
	for _, input := range inputs {
		if t.IsConsumed(input) {
			return ErrInputConsumed
		}
		if _, ok := seenInBatch[input]; ok {
			return ErrInputConsumed
		}
		seenInBatch[input] = struct{}{}
	}

	for _, input := range inputs {
		t.consumed[input] = struct{}{}
	}
	return nil
}

func newReplayCacheWithClock(defaultTTL int64, now func() int64) *ReplayCache {
	return &ReplayCache{
		defaultTTL: defaultTTL,
		now:        now,
		seen:       make(map[[16]byte]int64),
	}
}

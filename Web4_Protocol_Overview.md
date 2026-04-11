# Web4 Protocol Overview (Core Spec v1)

## Architecture Overview

```text
[ Crypto Layer ]
    ↓
[ Transfer / Identity Layer ]
    ↓
[ Message / Transport Layer ]
    ↓
[ Validation / Selection ]
    ↓
[ P2P Network ]
    ↓
[ Gossip Propagation ]
```

---

## Core Philosophy

Web4 is NOT a global consensus system.

* No global ledger
* No global state agreement
* No global finality

Instead:

> Each node independently validates, observes, and selects one lineage.

---

# 1. Crypto Stack

## Goals

* Integrity
* Authentication
* Optional confidentiality

---

## Required

* Hash: `BLAKE3` (preferred) or `SHA-256`
* Signature: `Ed25519`
* AEAD: `XChaCha20-Poly1305`

---

## Rules

* All IDs are **content-based hashes**
* Transfers MUST be signed
* Cryptographic identity is deterministic

---

# 2. Transfer Layer (Identity)

## Transfer

```go
type Transfer struct {
    ID      Hash
    Inputs  []Hash
    Outputs []Value
    Sig     []byte
}
```

---

## CRITICAL RULES

* Transfer ID = hash(canonical encoding)
* Canonical encoding includes:

  * Inputs
  * Outputs
* Canonical encoding EXCLUDES:

  * Timestamp
  * Signature
  * ID

---

## Meaning

> Transfer identity = “what happened”, not “when it happened”

---

## Signature

* Signature is computed over canonical transfer bytes
* Signature is NOT part of Transfer ID

---

# 3. Message Layer (Transport)

## Message Structure

```go
type Message struct {
    MessageID [16]byte
    Type      MsgType
    Payload   []byte
    Timestamp int64
    TTL       int64
}
```

---

## Message Rules

* MessageID = random per message
* Used ONLY for replay protection
* Message metadata MUST NOT affect transfer identity

---

## Layer Separation

| Layer    | Responsibility        |
| -------- | --------------------- |
| Transfer | value semantics       |
| Message  | network transport     |
| Payload  | message-specific data |

---

# 4. Message Types

```go
type MsgType int

const (
    INV MsgType = iota
    GET
    TX
    PROOF_REQ
    PROOF_RESP
    ACK
    ERROR
)
```

---

# 5. TX Payload (CRITICAL)

## Structure

```go
type TXPayload struct {
    TransferBytes []byte
    Signature     []byte
}
```

---

## Rules

* TransferBytes = canonical transfer encoding
* Signature = canonical transfer signature
* Transfer ID = hash(TransferBytes)

---

## Validation Flow

```text
decode TransferBytes
reconstruct transfer
attach Signature
verify signature
if invalid → reject
```

---

## IMPORTANT

* Signature is transported, NOT part of identity
* Timestamp is NOT part of identity

---

# 6. Validation (Local)

Each node validates independently:

```go
validate(tx):
  if signature invalid → reject
  if input unknown → reject or pending
  if input already consumed locally → reject
  if expiry passed → reject
  else → accept candidate
```

---

# 7. Conflict & Selection

## Conflict Set

Transactions sharing same input.

---

## Selection Function

```go
choose(tx_i):
  score = f(
    first_seen,
    propagation_count,
    time_stability,
    proof_weight,
    conflict_cleanliness
  )
  return max(score)
```

---

## Rule

> Each node selects ONE lineage per input.

---

# 8. Finality

## Confidence Model

```go
confidence(tx):
  = time_weight
  + propagation_weight
  + proof_weight
```

---

## Acceptance Rule

```go
if confidence > threshold:
    FINAL
else:
    pending
```

---

## Notes

* Threshold depends on amount
* No global finality exists

---

# 9. Gossip Propagation

## Flow

```text
A → B: INV(tx_id)
B → A: GET(tx_id)
A → B: TX(payload)
```

---

## Rules

* Never send TX blindly
* Use INV → GET → TX
* Maintain seen cache
* Use TTL pruning

---

# 10. Proof Protocol

## When Used

* Medium / high-value transactions
* Strengthens selection confidence

---

## Flow

```text
node → peers: PROOF_REQ(tx_id)
peer → node: PROOF_RESP
```

---

## Proof Response

```go
type ProofResponse struct {
    TxID       Hash
    Seen       bool
    SeenTime   int64
    Conflict   bool
    Distance   int
}
```

---

# 11. Replay Protection

## Mechanisms

* seen_message_ids
* seen_inputs

---

## Rules

* message_id → prevents message replay
* input → prevents local double spend

---

## IMPORTANT

Replay protection is:

* local
* message-layer only

---

# 12. Core Invariants

1. Deterministic identity (content-based hash)
2. Unit-wise conservation
3. Issuance ownership
4. Local input single-use
5. Local acceptance autonomy

---

# 13. Properties

## Advantages

* No global consensus required
* Scales naturally
* Resilient to partial failure
* Supports conflict-based model

---

## Trade-offs

* No global truth
* Eventual convergence only
* Network latency affects outcomes

---

# 14. Summary

> Web4 propagates transactions using gossip, where transfers are defined purely by their content and transmitted with signatures for verification.
> Each node independently validates and selects one lineage from competing transactions using local observations and neighborhood proofs, without requiring global consensus.

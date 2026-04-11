# Web4 P2P & Gossip Protocol v1

## Overview

This document defines the peer-to-peer network layer and message propagation model for Web4.

Web4 does not rely on global consensus.
Instead, it uses **gossip-based propagation** and **local validation + selection**.

---

## Design Goals

1. No global synchronization
2. Efficient propagation of transactions
3. Conflict-aware dissemination
4. Low bandwidth overhead
5. Replay resistance
6. Support for neighborhood-based validation

---

# 1. Network Model

## Node Types

| Type       | Role                  |
| ---------- | --------------------- |
| Edge Node  | User client           |
| Relay Node | Propagation / routing |
| Entry Node | Bootstrap only        |

---

## Transport

* QUIC (recommended)
* UDP-based
* Port: 443 preferred (firewall traversal)

---

## Connection Model

* Persistent peer connections
* Peer discovery via:

  * bootstrap nodes
  * peer exchange (PEX)

---

# 2. Message Types

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

## Message ID Rule

* Random per message
* Used for replay protection
* NOT related to transfer identity

---

# 3. Core Separation (IMPORTANT)

Web4 strictly separates layers:

| Layer    | Purpose               |
| -------- | --------------------- |
| Transfer | value semantics       |
| Message  | transport container   |
| Payload  | message-specific data |

---

## Critical Rules

* Transfer identity MUST NOT include:

  * Timestamp
  * Signature
* Message metadata MUST NOT affect transfer identity

---

# 4. Gossip Flow

## Basic Flow

```text
A → B: INV(tx_id)
B → A: GET(tx_id)
A → B: TX(payload)
```

---

## Rules

* Never send full TX blindly
* Always use INV → GET → TX
* Reduces bandwidth

---

# 5. TX Payload (UPDATED — CRITICAL)

## Structure

```go
type TXPayload struct {
    TransferBytes []byte
    Signature     []byte
}
```

---

## Rules

* `TransferBytes` = canonical transfer encoding
* `Signature` = canonical transfer signature

---

## IMPORTANT

* Transfer ID = hash(TransferBytes)
* Signature is NOT part of transfer identity
* Timestamp is NOT part of transfer identity

---

## Validation Flow

```text
decode TransferBytes
reconstruct transfer
attach Signature
verify canonical signature
if invalid → reject
```

---

# 6. INV Payload

```text
Payload = transfer_id (32 bytes)
```

---

# 7. GET Payload

```text
Payload = transfer_id (32 bytes)
```

---

# 8. Propagation Logic

## onReceive INV

```text
if tx_id already known:
    ignore
else:
    send GET(tx_id)
```

---

## onReceive TX

```text
if seen_message:
    ignore

decode TXPayload

if !validate(transfer + signature):
    reject

store tx
update conflict set
recompute selection

broadcast INV(tx_id)
```

---

# 9. Broadcast Strategy

## Gossip Rules

* Send to subset of peers (fanout)
* Randomized peer selection
* Avoid flooding entire network

---

## Fanout Example

```text
fanout = sqrt(peer_count)
```

---

# 10. Replay Protection

## seen_messages cache

* key: message_id
* TTL-based expiry

---

## Rule

```text
if message_id in seen:
    drop
```

---

## IMPORTANT

Replay protection is:

* message-level
* local-only

NOT part of transfer identity

---

# 11. Peer Observation

Each node tracks:

```go
map[tx_id]map[peer_id]timestamp
```

Used for:

* propagation score
* independence detection

---

# 12. Proof Protocol

## When used

* medium / high value transactions
* conflict resolution strengthening

---

## Flow

```text
A → peers: PROOF_REQ(tx_id)
peer → A: PROOF_RESP
```

---

## Proof Request

```go
type ProofRequest struct {
    TxID Hash
}
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

## Rule

* Aggregate responses
* Convert to proof weight
* Feed into selection function

---

# 13. Neighborhood Expansion

Used for high-value transactions.

---

## Expansion Strategy

| Tier   | Radius     |
| ------ | ---------- |
| Tier 0 | local only |
| Tier 1 | 1-hop      |
| Tier 2 | 2-hop      |
| Tier 3 | 3-hop+     |

---

## Trigger

```text
if amount > threshold:
    increase proof radius
```

---

# 14. Conflict Propagation

## Rule

* If node sees competing tx:

  * propagate ALL
  * do not filter

---

## Reason

> Selection is local, not global

---

# 15. Rate Limiting

To prevent spam:

* per-peer rate limits
* INV throttling
* TX size limits

---

# 16. Peer Exchange (PEX)

## Message

```go
type PeerExchange struct {
    Peers []PeerInfo
}
```

---

## Rule

* periodically share known peers
* avoid central discovery

---

# 17. Failure Handling

## Timeout

* GET timeout → retry other peer

## Invalid TX

* reject immediately
* optionally mark peer as suspicious

---

# 18. Security Considerations

### Known Risks

* Sybil attack
* propagation manipulation
* eclipse attack

---

### Mitigation (v1 basic)

* peer diversity
* connection limits
* random peer rotation

---

# 19. Properties

### Advantages

* no global broadcast requirement
* scalable
* resilient to partial failure
* supports conflict-based model

---

### Trade-offs

* eventual consistency only
* latency affects outcomes
* requires good peer selection

---

# 20. Summary

> Web4 uses gossip-based propagation via INV/GET/TX flows.
> Transactions are transmitted as canonical transfer bytes + signature, allowing remote validation without affecting transfer identity.
> Conflicts are propagated without suppression, and each node independently validates and selects lineage using local observations and neighborhood proofs.

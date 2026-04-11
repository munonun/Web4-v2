# Web4 Storage Architecture v1

## Overview

This document defines the local storage model required to support:

* conflict tracking
* lineage selection
* propagation awareness
* replay protection
* finality evaluation

All storage is **node-local**.
There is no global shared state.

---

## Storage Principles

1. No global ledger
2. Only locally observed data is stored
3. State is ephemeral where possible
4. Conflict is preserved, not erased
5. Selected lineage is explicitly stored

---

# 1. Core Storage Components

The node maintains the following storage layers:

| Component           | Purpose                          |
| ------------------- | -------------------------------- |
| `tx_store`          | All known transactions           |
| `conflict_sets`     | Competing transactions per input |
| `selected_lineage`  | Current chosen branch            |
| `input_consumed`    | Local single-use enforcement     |
| `seen_messages`     | Replay protection                |
| `peer_observations` | Propagation tracking             |
| `proof_store`       | Neighborhood proofs              |
| `finality_state`    | Confidence + status              |
| `expiry_index`      | TTL pruning                      |

---

# 2. tx_store

Stores all received transactions.

### Key

```id="key_tx"
tx_id
```

### Value

```go id="tx_store_struct"
type TxRecord struct {
    Tx          Transfer
    FirstSeen   int64
    SeenCount   int
    Peers       map[PeerID]bool
}
```

---

# 3. conflict_sets

Tracks competing transactions per input.

### Key

```id="conflict_key"
input_id
```

### Value

```go id="conflict_struct"
type ConflictSet struct {
    TxIDs []Hash
}
```

---

## Rule

* Every input_id maps to **multiple tx**
* Never delete conflicts immediately
* Selection happens on top of this

---

# 4. selected_lineage

Stores chosen tx per input.

### Key

```id="sel_key"
input_id
```

### Value

```go id="sel_struct"
type Selected struct {
    TxID      Hash
    Score     float64
    UpdatedAt int64
}
```

---

## Rule

* Only one tx per input is selected
* Updated whenever scores change

---

# 5. input_consumed

Enforces local single-use rule.

### Key

```id="input_key"
input_id
```

### Value

```go id="input_struct"
type Consumed struct {
    TxID Hash
}
```

---

## Rule

* If input already consumed → reject new tx
* Local only (not global truth)

---

# 6. seen_messages

Prevents replay.

### Key

```id="msg_key"
message_id
```

### Value

```go id="msg_struct"
timestamp
```

---

## Rule

* Expire after TTL
* Used before validation

---

# 7. peer_observations

Tracks propagation.

### Key

```id="peer_key"
tx_id
```

### Value

```go id="peer_struct"
type PeerObservation struct {
    Peers map[PeerID]int64 // peer → timestamp
}
```

---

## Rule

* Used for P(tx)
* Must avoid counting duplicate peers

---

# 8. proof_store

Stores neighborhood proofs.

### Key

```id="proof_key"
tx_id
```

### Value

```go id="proof_struct"
type ProofRecord struct {
    Proofs []Proof
}
```

---

### Proof Structure

```go id="proof_inner"
type Proof struct {
    PeerID      PeerID
    Distance    int     // hop count
    SeenTime    int64
    NoConflict  bool
}
```

---

# 9. finality_state

Stores confidence and status.

### Key

```id="final_key"
tx_id
```

### Value

```go id="final_struct"
type Finality struct {
    Score       float64
    Confidence  float64
    Status      string // PENDING / SELECTED / FINAL
}
```

---

# 10. expiry_index

Tracks expiration.

### Key

```id="exp_key"
expiry_time
```

### Value

```go id="exp_struct"
[]tx_id
```

---

## Rule

* Periodically prune:

  * expired tx
  * old conflict sets
  * stale proofs

---

# 11. State Transitions

## onReceive(tx)

```text
1. check seen_messages
2. validate(tx)
3. insert into tx_store
4. update conflict_sets
5. update peer_observations
6. recompute selection
7. update selected_lineage
8. update finality_state
```

---

## onConflictUpdate(input_id)

```text
1. get conflict set
2. compute score(tx) for all
3. select best tx
4. update selected_lineage
5. update finality_state
```

---

## onFinalityCheck(tx)

```text
if Confidence(tx) >= threshold:
    mark FINAL
```

---

# 12. Pruning Strategy

## Must prune:

* expired tx
* inactive conflicts
* old proofs
* seen_messages cache

---

## Must NOT prune:

* active selected lineage
* recent conflicts

---

# 13. Storage Properties

### Advantages

* fully local
* no global sync
* supports DAG conflict model
* scalable

---

### Trade-offs

* memory growth (needs pruning)
* conflict sets can grow
* proof storage overhead

---

# 14. Minimal Implementation Requirement

To function correctly, node MUST implement:

* tx_store
* conflict_sets
* selected_lineage
* input_consumed
* peer_observations

Everything else can be incremental.

---

# 15. Summary

> Web4 storage does not maintain a global ledger.
> Instead, it stores locally observed transactions, conflict sets, and selected lineage states.
> Each node maintains its own view of the conflict graph and derives its own finality.

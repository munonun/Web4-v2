# Web4 — Local Selection Value Network

Web4 is a distributed value-transfer protocol where nodes validate locally, preserve conflicts, and select one lineage per input without global consensus.

## Core Idea

Web4 does not maintain a global ledger.

There is no global consensus process and no single network-wide state that every node must agree on before progress can happen. Each node validates transfers with its own local view, stores conflicting transactions instead of deleting them, and selects exactly one lineage per input using a deterministic scoring function.

Conflicts are allowed to exist. Convergence emerges because nodes tend to see similar propagation, stability, and proof signals over time, not because they are forced into a single shared truth at every step.

## Architecture Overview

Web4 is built in layers.

- Crypto
  - Canonical binary encoding for values and transfers
  - Content-based transfer IDs
  - Ed25519 signatures
  - Message IDs and AEAD support
- Transfer
  - Deterministic transfer identity based on canonical transfer bytes
  - Timestamp excluded from transfer identity
  - Signature verified over canonical transfer content
- Message
  - Envelope with `MessageID`, `Timestamp`, `TTL`, `Type`, and `Payload`
  - Deterministic payload encodings for `INV`, `GET`, `TX`, `PROOF_REQ`, `PROOF_RESP`, `ACK`, and `ERROR`
- Node runtime
  - Local tx store, conflict sets, selected lineage, proof store, finality state
  - Replay protection and local input tracking
  - Selection scoring and amount-tiered finality
- Gossip
  - `INV -> GET -> TX` flow
  - Proof request / response flow for stronger confidence on higher-value transfers

## Quick Start

Run the full deterministic demo suite:

```bash
./scripts/demo_all.sh
```

This runs three simulations in order:

- `basic`
  - shows propagation through `INV -> GET -> TX`
- `conflict`
  - shows two conflicting transfers competing for the same input
- `proof`
  - shows proof responses increasing confidence and changing finality

## Demo Breakdown

### demo basic

This demo creates three in-memory nodes: `A`, `B`, and `C`.

Node `A` creates a valid transfer and gossips it to the network. `B` and `C` receive `INV`, request `GET`, receive `TX`, validate locally, and update their selected state.

Run it with:

```bash
./scripts/demo_basic.sh
```

### demo conflict

This demo creates two different transfers that spend the same input.

All nodes receive both transactions, keep the conflict set, compute scores, and select one lineage deterministically.

Run it with:

```bash
./scripts/demo_conflict.sh
```

### demo proof

This demo creates a higher-value transfer and then simulates `PROOF_REQ` / `PROOF_RESP` messages from peers.

The node updates proof weight, recomputes confidence, and may advance the transfer from `SELECTED` to `FINAL`.

Run it with:

```bash
./scripts/demo_proof.sh
```

## Example Output

Shortened example output from the current CLI:

```text
=== BASIC ===
[A] created tx: 1500790bf766
[A -> B] INV 1500790bf766
[B -> A] GET 1500790bf766
[A -> B] TX 1500790bf766
[B] accepted tx 1500790bf766

=== CONFLICT ===
Conflict detected on input 046cd2ce56e1:
  tx: 5e246b0eaacc
  tx: f5f6583a45a3

Selected:
  Node A -> f5f6583a45a3
  Node B -> f5f6583a45a3

=== PROOF ===
Proof responses:
  Peer B -> seen=true conflict=false distance=1 seen_time=100
  Peer C -> seen=true conflict=false distance=1 seen_time=100

Confidence:
  before = 0.21
  after  = 0.79

Finality: FINAL
```

## Key Properties

- no global consensus
- local state only
- deterministic content-based IDs
- conflict tolerance instead of conflict suppression
- convergence through shared local signals over time

## What This Is Not

Web4 is not a blockchain.

It does not produce a single globally ordered ledger. It does not assume every node will hold the same state at the same moment. It does not provide instant finality or a universal notion of truth.

The system is designed around local validation, local selection, and practical convergence.

## Roadmap

Current code is an in-memory prototype focused on protocol behavior.

Likely next steps:

- real P2P transport, including QUIC-based networking
- persistent local storage for tx state, conflict sets, proofs, and finality
- fuller CLI support for starting nodes and submitting transfers
- richer proof collection and aggregation
- adversarial and failure-mode testing

## Development

Run tests:

```bash
go test ./...
```

Run all demos:

```bash
./scripts/demo_all.sh
```

The repository also includes focused demo scripts:

```bash
./scripts/demo_basic.sh
./scripts/demo_conflict.sh
./scripts/demo_proof.sh
```

## Philosophy

Most distributed systems start by asking how all nodes can agree on one global state.

Web4 starts from a different premise: nodes do not need a single global truth to make progress. They need deterministic local rules, conflict preservation, and enough overlap in observation that similar nodes usually select the same lineage.

This is the difference between consensus and selection.

Consensus tries to force one answer everywhere.

Selection lets each node decide locally, then relies on propagation, stability, and proof signals to make those local decisions converge often enough to be useful.

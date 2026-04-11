# AGENTS.md

## Scope

- This repo is still spec-led, but it now also contains a real Go package at `crypto/` implementing the protocol crypto layer. Most other root files remain Markdown specs.
- The only runnable verification currently in-repo is Go-based: `go test ./...` runs the package tests, and `go test -bench . ./crypto` runs the hash benchmarks.
- There is still no CI workflow, task runner, or workspace config. Cross-document consistency matters because the Markdown specs remain the broader protocol source of truth.

## Canonical Docs

- `Web4_Protocol_Overview.md` is the cross-cutting summary of the model and its invariants.
- `Web4_P2P&Gossip-Protocol.md` is the authority for transport, node roles, message types, gossip flow, proofs, PEX, and network security/failure behavior.
- `Web4_Selection_Function.md` is the authority for the validation gate, score components, tie-break order, confidence, and amount-tiered finality.
- `Web4_Storage_Architecture.md` is the authority for node-local state, replay cache, proof storage, selected lineage, and pruning/state transitions.
- `selection_convergence.md` is a rationale document for why local selection converges; update it after the canonical spec docs when selection/finality behavior changes.
- `crypto/` is the executable source of truth for hashing, canonical binary encoding, content-derived IDs, Ed25519 signing, XChaCha20-Poly1305 AEAD, and message ID generation.

## Non-Obvious Constraints

- Preserve the Web4 model: no global consensus, no global ledger, and no suppression of conflicts. Nodes validate locally, preserve conflict sets, and select exactly one lineage per input.
- Treat the detailed domain docs as authoritative when the overview is shorter or stale. The overview already drifts in duplicated areas.
- Shared definitions are easy to partially update. Hot drift points are message enum/flow names (`TRANSFER` in the overview vs `TX` in the P2P spec), message fields, proof request/response structures, replay rules, finality tiers/thresholds, and deterministic tie-break order.
- Storage is explicitly node-local and conflict-preserving. Do not rewrite the spec toward blockchain-style global state or immediate deletion of losing conflicts.

## Edit Checks

- Crypto-layer changes: keep `crypto/` aligned with `Web4_Protocol_Overview.md`, especially hashing choices, canonical encoding rules, content-based ID rules, Ed25519 signatures, XChaCha20-Poly1305, and message ID behavior.
- Networking, message, or proof changes: update `Web4_Protocol_Overview.md`, `Web4_P2P&Gossip-Protocol.md`, and `Web4_Storage_Architecture.md` together.
- Scoring, tie-break, or finality changes: update `Web4_Protocol_Overview.md`, `Web4_Selection_Function.md`, `Web4_Storage_Architecture.md`, and `selection_convergence.md` if its rationale/examples mention the changed rule.
- After renaming any shared concept, grep all root `*.md` files; there is no executable source of truth to catch drift for you.
- Keep prose, tables, examples, pseudocode, and the Go implementation aligned in the same edit.

# Web4 Selection Function v1

## Overview

The Web4 selection function defines how each node deterministically selects a single lineage from conflicting transactions.

Unlike consensus-based systems, Web4 allows multiple conflicting transactions to exist, but each node selects exactly one based on a local scoring function.

---

## Goals

The selection function must:

1. Select one transaction from a conflict set
2. Support different finality strength based on transaction amount
3. Operate deterministically per node (no global coordination)
4. Incorporate time, propagation, and proof signals

---

## Definitions

### Input

A reference to an existing value/state.

Example:

```
X
```

---

### Conflict Set

A set of transactions that consume the same input.

Example:

```
tx1: X → Z1
tx2: X → Z2
```

```
ConflictSet(X) = {tx1, tx2}
```

---

### Node Selection

Each node `N` selects exactly one transaction per conflict set:

```
f_N(X) = tx_i
```

---

## Transaction States

Each transaction has one of the following local states:

* `PENDING`: valid but not selected
* `SELECTED`: currently chosen within conflict set
* `FINAL`: confidence exceeds threshold

---

## Validation Gate

Before entering selection, a transaction must pass:

* Valid signature
* Valid canonical hash
* All inputs known or resolvable
* Unit-wise conservation holds
* Not expired
* No malformed proof
* Not a replayed message

Invalid transactions are rejected immediately.

---

## Scoring Function

Each transaction is assigned a score:

```
Score(tx) =
  w_f * F(tx) +
  w_p * P(tx) +
  w_t * T(tx) +
  w_n * N(tx) +
  w_c * C(tx)
```

---

### Components

#### F(tx): First-Seen Score

```
F(tx) = 1 / (1 + rank(tx))
```

* rank = order of arrival within conflict set
* first seen = highest score

---

#### P(tx): Propagation Score

```
P(tx) = log(1 + peerCount(tx))
```

* peerCount = number of peers that independently reported the tx

---

#### T(tx): Time Stability Score

```
T(tx) = min(1, age(tx) / τ)
```

* age = time since first seen
* τ = stabilization constant (e.g. 30 seconds)

---

#### N(tx): Neighborhood Proof Score

```
N(tx) = proofWeight(tx) / (proofWeight(tx) + k)
```

* proofWeight = aggregated neighborhood proof score
* k = smoothing constant

Example weights:

* 1-hop: 1.0
* 2-hop: 0.6
* 3-hop: 0.3

---

#### C(tx): Conflict Cleanliness Score

```
C(tx) = 1 - (conflictSignals(tx) / (maxSignals + 1))
```

* fewer conflict signals = higher score

---

## Recommended Weights

```
w_f = 0.35
w_p = 0.20
w_t = 0.20
w_n = 0.20
w_c = 0.05
```

Interpretation:

> Prefer transactions that were seen early, propagated widely, remained stable, and have strong neighborhood support.

---

## Selection Rule

```
selected(X) = argmax Score(tx) for tx ∈ ConflictSet(X)
```

---

## Tie-Break Rules

If scores are equal, resolve in order:

1. Higher F(tx)
2. Higher P(tx)
3. Higher T(tx)
4. Higher N(tx)
5. Lexicographically smaller tx ID

---

## Confidence Function

Used to determine finality:

```
Confidence(tx) =
  α_p * P(tx) +
  α_t * T(tx) +
  α_n * N(tx)
```

Recommended:

```
α_p = 0.30
α_t = 0.35
α_n = 0.35
```

---

## Finality Thresholds

Finality depends on transaction amount:

| Tier   | Threshold |
| ------ | --------- |
| Small  | 0.45      |
| Medium | 0.70      |
| Large  | 0.88      |

---

### Finality Rule

```
if selected(tx) and Confidence(tx) < threshold:
    status = SELECTED

if selected(tx) and Confidence(tx) ≥ threshold:
    status = FINAL
```

---

## Neighborhood Finality

Validation radius increases with transaction size:

| Tier   | Condition        | Requirement     |
| ------ | ---------------- | --------------- |
| Tier 0 | amount < T1      | local only      |
| Tier 1 | T1 ≤ amount < T2 | 1-hop proofs    |
| Tier 2 | T2 ≤ amount < T3 | 2-hop proofs    |
| Tier 3 | amount ≥ T3      | extended proofs |

Example:

```
T1 = 50,000
T2 = 500,000
T3 = 3,000,000
```

---

## Local Algorithm

```
onReceive(tx):
  if !validate(tx):
      reject

  insert tx into ConflictSet(inputs)

  recompute scores for all tx in set

  selected = maxScore(conflictSet)

  if tx == selected:
      if Confidence(tx) >= threshold(amount):
          mark FINAL
      else:
          mark SELECTED
  else:
      mark PENDING
```

---

## Properties

### Advantages

* Fully local and deterministic
* No global consensus required
* Supports tiered finality
* Integrates propagation and proof signals

---

### Limitations

* Network latency influences outcomes
* Vulnerable to propagation manipulation
* Requires Sybil resistance improvements
* Weight tuning required

---

## Future Improvements (v2)

* Independent peer counting
* Cluster diversity weighting
* Sybil resistance mechanisms
* Adaptive thresholds
* Anti-first-seen manipulation

---

## Summary

> Web4 does not eliminate conflicting transactions.
> Instead, each node deterministically selects one lineage using local observations, propagation signals, and neighborhood proofs.
> Finality is achieved when confidence exceeds an amount-dependent threshold.

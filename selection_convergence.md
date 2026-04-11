# Web4 Selection Convergence v1

## Overview

This document explains how Web4, despite having no global consensus, can still converge enough to behave like a practical value-transfer system.

Web4 does not force a single global truth.
Instead, it attempts to make most nodes independently choose the same lineage under normal conditions.

This is called **practical convergence**.

---

## 1. Why Convergence Matters

Without convergence, Web4 cannot behave like money.

If different nodes keep selecting different conflicting transactions for the same input, then:

* balances diverge too much
* payment finality becomes unclear
* the same unit cannot be trusted across the network
* users stop perceiving the system as a usable medium of exchange

In other words:

> Web4 does not need global consensus, but it does need most nodes to usually make similar choices.

---

## 2. What Prevents Convergence

The following factors push nodes toward different outcomes.

### 2.1 Network Latency Differences

Some nodes receive a transaction earlier than others.

If first-seen order dominates too strongly, then local timing differences can cause different selections.

---

### 2.2 Conflict Propagation Delay

If conflicting transactions spread unevenly, some nodes may see only one branch while others see both.

This creates temporary divergence.

---

### 2.3 Poor Peer Diversity

If a node is connected to a narrow or biased peer set, then its local view can be distorted.

This increases the chance of selecting a minority lineage.

---

### 2.4 Weak Tie-Break Rules

If two transactions receive similar scores and the final tie-break is ambiguous, different nodes may choose differently.

---

### 2.5 High-Value Low-Visibility Transactions

For larger transfers, purely local observation may be too weak.

Nodes may need wider neighborhood evidence before converging on the same branch.

---

## 3. What Produces Convergence

Web4 convergence emerges from multiple signals working together.

### 3.1 Early Visibility Matters, But Does Not Dominate

A transaction seen earlier gains an initial advantage.

However, first-seen order alone must not determine the outcome in all cases.

This prevents local latency from becoming the only source of power.

---

### 3.2 Wider Propagation Increases Selection Probability

A transaction seen by more independent peers is more likely to be selected.

This creates pressure toward branches that spread broadly rather than branches that only arrive early in one local view.

---

### 3.3 Time Stability Rewards Surviving Branches

If a transaction remains conflict-free for longer, confidence increases.

This helps suppress short-lived or weak branches.

---

### 3.4 Neighborhood Proof Strengthens Local Certainty

For larger transactions, nodes can gather evidence from nearby peers.

This reduces the chance that isolated local views determine important outcomes.

---

### 3.5 Deterministic Tie-Break Ensures Last-Step Alignment

When scores are close or identical, every node must resolve ties the same way.

This is critical.

Without deterministic tie-break, small scoring differences would create persistent fragmentation.

---

## 4. Role of Each Selection Score Component

The selection score is not just a ranking function.
It is the mechanism that pushes nodes toward similar choices.

### 4.1 First-Seen Score

Purpose:

* gives an initial advantage to earlier transactions
* helps reduce indecision in conflict sets

Risk:

* over-weighting it makes latency too powerful

---

### 4.2 Propagation Score

Purpose:

* rewards transactions observed by more peers
* helps the network favor branches that spread broadly

Why it matters:

* this is one of the strongest convergence signals

---

### 4.3 Time Stability Score

Purpose:

* increases confidence for branches that remain alive over time
* reduces the success rate of unstable conflicts

Why it matters:

* convergence is not only about who was first, but who persists

---

### 4.4 Neighborhood Proof Score

Purpose:

* adds evidence from nearby nodes
* improves alignment for higher-value transactions

Why it matters:

* for larger payments, local observation alone is often too weak

---

### 4.5 Conflict Cleanliness Score

Purpose:

* penalizes branches that attract conflict signals
* helps eliminate noisy or suspicious branches

Why it matters:

* reduces acceptance of branches that look unstable

---

## 5. Deterministic Tie-Break

Tie-break must be fixed and deterministic.

Recommended order:

1. Higher first-seen score
2. Higher propagation score
3. Higher time stability score
4. Higher neighborhood proof score
5. Lexicographically smaller transaction ID

This ensures:

* same local inputs produce same output
* nodes do not split randomly
* convergence is preserved even when scores are close

---

## 6. Why Neighborhood Proof Is Needed for High-Value Transactions

Small payments can often rely on local observation plus propagation.

High-value payments are different.

For larger transactions:

* local visibility may be incomplete
* conflict costs are higher
* temporary divergence becomes more dangerous

Therefore, high-value transfers should expand validation radius.

### Tiered Model

* Low value: local selection only
* Medium value: 1-hop neighborhood proof
* High value: 2-hop neighborhood proof
* Very high value: wider neighborhood proof

This preserves Web4’s local nature while increasing convergence pressure for important transfers.

---

## 7. Conditions Under Which Web4 Can Look Like Money

Web4 does not become money by decree.
It begins to look like money when users experience stable, repeatable outcomes.

That requires the following conditions.

### 7.1 Most Nodes Usually Select the Same Lineage

Practical convergence must be high enough that divergence is rare in normal use.

---

### 7.2 Conflicts Become Hard to Notice

Double-spend attempts may still exist, but users should rarely see unresolved conflict outcomes.

---

### 7.3 Confidence Rises Quickly

For ordinary payments, confidence should grow fast enough that users perceive transactions as practically final.

---

### 7.4 Acceptance Becomes Broad

The more participants accept the same units and lineages, the more the system behaves like money.

---

### 7.5 Risk Remains Tolerable

The system does not need zero risk.
It needs risk low enough that users continue using it.

---

### 7.6 Payment Policy Is Stable Across Value Tiers

Small, medium, and high-value payments should each have predictable acceptance behavior.

---

### 7.7 User Experience Hides Internal Complexity

Users should not need to understand lineage, conflict graphs, or proof collection.

They should only experience:

* accepted
* pending
* final

---

## 8. Practical Convergence Statement

Web4 does not seek universal consensus.

Instead, it seeks the following condition:

> Under normal network conditions, honest transactions should propagate widely, accumulate stability, and gather enough local and neighborhood evidence that most nodes independently select the same lineage.

This is the core convergence target.

---

## 9. Summary

Web4 avoids global consensus, but it still needs convergence.

That convergence does not come from forcing all nodes into one shared state.
It comes from making the same branch naturally win in most local views.

This happens when:

* transactions propagate broadly
* unstable branches lose over time
* higher-value transfers gather neighborhood proof
* tie-break rules are deterministic

In short:

> Web4 behaves like money not when everyone agrees by force, but when most nodes naturally keep choosing the same branch.

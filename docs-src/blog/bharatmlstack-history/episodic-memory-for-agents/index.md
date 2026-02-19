---
title: "Beyond Vector RAG: Building Agent Memory That Learns From Experience."
description: "Current agent memory is just search. We built an episodic memory system that tracks outcomes, forms causal links, extracts reasoning heuristics, and actually learns from failure — without retraining the model."
slug: episodic-memory-for-agents
authors: [adarsha]
date: 2026-02-19
tags: [ai-agents, memory, architecture, llm, episodic-memory]
---
![BharatMLStack](./bms.png)
Every agent framework on the market will tell you their agents "have memory." What they mean is: they have a vector database.

They chunk text, embed it, store it, and retrieve whatever looks similar at query time. This works for document Q&A. It fails the moment you expect an agent to recall what happened last time, learn from a mistake, or avoid repeating a failed approach.

We are trying to built something different. An episodic memory system where a frozen LLM — same weights, no retraining — produces increasingly better decisions over time because the memory feeding it context is continuously evolving.

Then we tested it. The results surprised us.

<!-- truncate -->

## The Gap Nobody Talks About

Here's a scenario every engineering team has encountered: AI agent hits a Redis connection pool exhaustion issue. It misdiagnoses it as a database problem. You correct it. Next week, a different service has the exact same failure pattern. The agent makes the exact same mistake.

Why? Because LLMs don't learn at inference time. Corrections adjust behavior within a conversation. Once the session ends, the lesson is gone. The model weights haven't changed. The next conversation starts from zero.

Current "memory" systems don't fully address this. They store facts — user preferences, document chunks, conversation summaries. But facts aren't experience. Knowing that "Redis connection pools can exhaust under load" is different from remembering "last time I saw 500 errors under load, I assumed it was the database, I was wrong, it was actually the connection pool, and here's the correction I received."

The first is a fact. The second is an episode. The difference matters.

## What's Wrong With Vector RAG as Memory

We identified five structural gaps in how current agent frameworks handle memory:

**No concept of time.** Two events are either semantically similar or they're not. The system can't represent "this happened after that" without distorting similarity scores. An agent can't reason about sequence or causality.

**No concept of situation.** A production incident and a design review might use the same technical vocabulary. Flat vector search can't distinguish them. Your agent retrieves planning notes when it should be retrieving incident postmortems.

**No outcome tracking.** The system stores *what happened* but not *whether it worked*. A failed approach and a successful one are equally retrievable. The agent has no way to prefer strategies that worked over strategies that didn't.

**Summaries destroy evidence.** Summarization-based memory compresses experience but discards the reasoning chain. The agent loses the ability to explain *how* it arrived at a conclusion. The audit trail is gone.

**No causal links.** Each memory chunk is independent. There's no way to express that incident A caused decision B, which led to outcome C, which was corrected by approach D. Without this structure, the agent can't traverse chains of reasoning.

These gaps compound. As an agent accumulates more experience, flat vector memory gets noisier, more contradictory, and less useful. The system degrades precisely when it should be improving.

## The Architecture: Episodic Memory

We are building a memory system modeled on how human episodic memory works — not as a metaphor, but as an engineering specification.

The system has four layers:

### Layer 1: Immutable Timeline

Every piece of agent experience is recorded as an append-only timeline entry. Each entry carries a semantic embedding (what it means), a timestamp (when it happened), and a state label (what situation the agent was in — debugging, planning, code review, incident response). Entries are never modified, never deleted, never summarized. This is the source of truth.

### Layer 2: Episode Segmentation

The system watches the timeline and detects when one coherent unit of experience ends and another begins — via state transitions, semantic shifts, temporal gaps, or explicit signals. Each episode is a reference into the timeline (not a copy) with a generated summary, an outcome (SUCCESS, FAILURE, PARTIAL, UNKNOWN), decisions made, assumptions held, and corrections received.

The outcome field is the most important thing that doesn't exist in any current memory system. Without it, you can't learn from mistakes.

### Layer 3: Episodic Graph

Episodes are connected through typed, weighted links: CAUSED_BY, LED_TO, RETRY_OF, LEARNED_FROM, CONTINUATION, CONTRADICTED. Over time, this forms a directed graph that enables traversal by meaning and causality. You can follow the chain: "this incident caused that investigation, which led to a failed fix, which was corrected by this approach."

### Layer 4: Generalized Facts

When multiple episodes exhibit consistent patterns, the system extracts reasoning heuristics: "When services fail immediately after deployment with no traffic change, investigate configuration errors before connection pool problems." Facts are versioned, never overwritten, and maintain links back to supporting and contradicting episodes. When contradicting evidence accumulates, confidence decreases. When confidence drops below a threshold, the fact is revised — but the old version is preserved.

The LLM sits above all four layers. At query time, the system assembles structured context — relevant episodes with outcomes, applicable facts with confidence scores, causal narratives — and passes it to the LLM for reasoning. The model reasons over structured memory. It doesn't store or manage memory.

### The Reinforcement Loop

This is where it comes together:

1. Agent reasons using retrieved episodes and facts
2. Outcome is detected (CI pass/fail, user correction, test result)
3. New episode is created with outcome tracking
4. Links are created between the retrieved episodes and the new episode
5. Facts are reinforced (if outcome aligned) or contradicted (if outcome conflicted)
6. If the decision was wrong and corrected, a LEARNED_FROM link is created

The model weights never change. The memory structure evolves continuously. A frozen LLM produces better decisions over time because it receives better context from richer memory.

## The Experiment

We built the full system in Python (~1,000 lines) and tested it head-to-head against a baseline flat-vector RAG agent across a 9-round synthetic debugging scenario. Both agents used the identical LLM (Claude Sonnet 4) for reasoning. The only variable was the memory system.

The scenario was designed to test five capabilities:

| Round Type | What It Tests | Rounds |
|---|---|---|
| LEARN | Can the agent build experience from failures? | 1, 2, 4 |
| RED HERRING | Can the agent resist applying a pattern when it doesn't fit? | 3 |
| TEST | Can the agent apply learned patterns to new services? | 5, 6 |
| SUBTLE | Can the agent generalize to different symptoms, same root cause? | 7 |
| CORRECTION | After being corrected, does the agent adapt? | 8, 9 |

Rounds 1-4 build experience: three connection pool failures across different services, plus one red herring (a deployment config error that *looks* like a connection pool issue). Rounds 5-7 test whether the agent applies the learned pattern to unfamiliar services and subtle symptom variations. Rounds 8-9 are the critical test: the agent is corrected after misdiagnosing a deployment-correlated error, then tested on a near-identical scenario to see if it adapts.

## Results

### Decision Accuracy

| Round | Type | Episodic Agent | Baseline Agent |
|---|---|---|---|
| 1 | LEARN | ✗ | ✓ |
| 2 | LEARN | ✓ | ✓ |
| 3 | RED HERRING | ✗ | ✗ |
| 4 | LEARN | ✓ | ✓ |
| 5 | TEST | **✓** | ✗ |
| 6 | TEST | **✓** | ✗ |
| 7 | SUBTLE | **✓** | ✗ |
| 8 | CORRECTION | ✓ | ✓ |
| 9 | CORRECTION | ✓ | ✓ |
| **Total** | | **7/9 (78%)** | **5/9 (56%)** |

The episodic agent won 7-5. A 40% relative improvement in decision accuracy using the exact same LLM.

### Where the Gap Opened

The episodic agent's advantage concentrated in exactly the rounds designed to test memory quality:

**Rounds 5-6 (pattern application):** The episodic agent cited 4 past failure episodes with connection pool exhaustion as root cause, complete with correction annotations. It correctly identified pool exhaustion in new services. The baseline retrieved disconnected chunks and suggested checking timeout configurations — a pattern it picked up from the Round 3 red herring.

**Round 7 (subtle symptoms — latency increase, no errors):** Both agents had the same evidence available. The episodic agent's retrieval surfaced a diverse set of episodes (thanks to MMR diversity filtering) including the Redis pool exhaustion from Round 6, which primed it to recognize that latency without errors can still be pool contention. The baseline defaulted to "check recent config changes."

**Round 9 (adaptation after correction):** This is the result we're most proud of. Look at the episodic agent's reasoning:

> *"Episode 1 directly parallels this situation — errors spiking immediately after a deployment (v2.4.1 then, v3.1.0 now) with no traffic change. In that case, the root cause was a database migration that dropped an index. The generalized fact confirms that deployment-related issues with immediate onset after version changes are more likely caused by configuration errors or missing dependencies than by connection pool problems."*

It cited a specific past episode by analogy, quoted a generalized fact, and explained *why* this situation matches the deployment pattern rather than the connection pool pattern. The baseline gave a vaguer assessment.

### Retrieval Quality

This is where the structural difference is most visible:

| Metric | Episodic Agent | Baseline Agent |
|---|---|---|
| Retrieved items with explicit outcome labels | **100%** | 25% |
| Correct pattern applications (Rounds 4-7) | **4/4** | 1/4 |
| False positives (Rounds 8-9) | **0** | 0 |

Every item the episodic agent retrieved carried a structured outcome label (SUCCESS or FAILURE) with correction details. Only 25% of the baseline's chunks contained any outcome information — and those were incidental text mentions, not structured labels.

The episodic agent correctly applied the connection pool pattern in all four rounds where it was the root cause, and correctly avoided it in both rounds where it wasn't. The baseline applied it correctly once.

## What Didn't Work

Two things didn't work as anticipated:

**Round 3 (red herring):** Both agents failed. The symptoms looked like connection pool issues, but the root cause was a deployment config change. At this point, the episodic agent had only seen connection pool episodes — it had no counter-evidence for deployment-correlated errors. You can't distinguish patterns you've only seen one side of. After Round 8 introduced a correction, the agent successfully avoided this mistake in Round 9.

**Fact quality variance.** Some extracted facts were specific and actionable ("Deployment-related issues with immediate onset are more likely configuration errors"). Others were vague ("Initial symptom-based diagnosis often leads to misidentifying the root cause"). A production system needs a usefulness filter, not just a confidence score.

## What This Means

The most important finding isn't the accuracy improvement. It's that the reinforcement loop closes without retraining.

In the POC, we observed:

- Rounds 1-4: Agent encounters failures, episodes recorded with outcomes and corrections
- After Round 4: Fact extracted — "Connection pool exhaustion is a common root cause under load"
- Rounds 5-7: Agent applies the pattern with increasing confidence (fact support count grows)
- Round 8: Agent encounters a deployment error, correctly identifies it as config, gets corrected
- After Round 8: New fact — "Deployment-related issues with immediate onset are more likely configuration errors"
- Round 9: Agent receives near-identical scenario, correctly avoids connection pool pattern, cites the Round 8 correction

The model didn't change. The memory evolved. That's the whole point.

## How It Compares to Existing Solutions

Agent memory is a fast-moving space with several strong systems, each solving a different slice of the problem:

**Mem0** excels at persistent personalization — extracting user preferences, managing session context, and reducing token costs through intelligent compression. It's the most production-ready memory layer available and integrates with nearly every agent framework. Its focus is on remembering about users and conversations rather than learning from task-level outcomes, which is a different problem than the one we're exploring here.

**Zep/Graphiti** is doing some of the most interesting work in temporal knowledge graphs. Their bi-temporal model — tracking both when an event occurred and when it was ingested — addresses a real structural gap in how agent memory handles changing facts over time. Their episode and entity subgraphs share some philosophical DNA with our approach. Where our work diverges is in outcome tracking and reinforcement: we're specifically focused on whether a decision worked, and using that signal to update memory structure.

**Letta (formerly MemGPT)** pioneered self-editing memory — giving the LLM tools to manage its own memory blocks. This is a powerful paradigm, and their recent work on "Context Repositories" and sleep-time compute suggests they're actively pushing toward agents that learn over time. Their team has been transparent that experiential learning is an unsolved problem, which is part of what motivated our exploration.

**MemRL (Jan 2026 paper)** is the closest to our work academically. It shares the core insight of decoupling stable LLM reasoning from plastic, evolving memory. Their approach uses reinforcement learning to assign utility Q-values to memories, which is elegant but requires training a value function. Our approach is purely structural — no training step, no Q-values, just graph evolution and LLM-based reasoning over outcomes.


The common thread: most existing systems focus on knowledge persistence — remembering facts, preferences, and conversation history across sessions. The problem we're exploring is experiential learning — tracking whether past decisions worked, forming causal chains between episodes, and extracting reasoning heuristics that improve over time. These are complementary capabilities that would be needed by an ideal production system.

## Try It Yourself

The prototype is available in our experiments directory:

```
experiments/episodic-memory-prototype/
├── memory/          # Timeline, encoder, episodes, graph, facts, retriever, reinforcer
├── agent/           # Episodic memory agent
├── baseline/        # Flat vector RAG agent (comparison)
├── simulator/       # 9-round debugging scenario
├── eval/            # Head-to-head comparison + scoring
└── tests/
```

To run the comparison:

```bash
cd experiments/episodic-memory-prototype
python -m venv .venv && source .venv/bin/activate
pip install -r requirements.txt
export ANTHROPIC_API_KEY=sk-ant-...
python -m eval.compare
```

Without an API key, it runs in heuristic mode (keyword-based decisions). With a key, both agents use Claude Sonnet for reasoning — that's where the quality gap becomes visible.


## Conclusion
This is a 9-round synthetic scenario we designed. It demonstrates the poc architecture works end-to-end and shows where episodic memory provides qualitatively different reasoning. It is not a peer-reviewed benchmark and should not be interpreted as a statistically rigorous claim. We're publishing the prototype so others can reproduce and extend the evaluation.
If this sparks interest do trigger github discussion.

---

*The episodic memory prototype is available in `BharatMLStack` repo at `/experiments/episodic-memory-prototype`*

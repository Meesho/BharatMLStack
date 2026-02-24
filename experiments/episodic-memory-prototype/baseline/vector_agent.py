"""Flat vector RAG agent -- baseline for comparison.

Stores every entry embedding in a flat list and retrieves the top-k most
similar items at query time.  No episodes, no graph, no facts, no
reinforcement.  This is the "standard RAG" baseline.

When ANTHROPIC_API_KEY is set, investigation decisions are made by Claude
using the raw text chunks.  Falls back to keyword heuristics when no key
is available.
"""

from __future__ import annotations

import os
from typing import Any

from memory.encoder import Encoder
from memory.models import (
    DecisionResult,
    ROLE_AGENT,
    ROLE_USER,
    TimelineEntry,
    cosine_similarity,
)

_POOL_PATTERNS = [
    "connection pool", "pool exhaust", "pool saturat",
    "pool starvation", "pool contention",
]

_INVESTIGATION_INSTRUCTION = (
    "Based on your past experience shown above, what should be investigated "
    "FIRST? Be specific. Explain your reasoning citing the past episodes "
    "and facts.\n\n"
    "Respond in this exact format:\n"
    "REASONING: <2-4 sentences analyzing the pattern from past entries>\n"
    "DECISION: Investigate: <one specific sentence>"
)


class VectorAgent:
    """Agent using a simple flat vector store (no episodic structure)."""

    def __init__(
        self,
        encoder: Encoder | None = None,
        top_k: int = 5,
        anthropic_client: Any = None,
        model: str = "claude-sonnet-4-20250514",
    ) -> None:
        self.encoder = encoder or Encoder()
        self.top_k = top_k
        self._entries: list[TimelineEntry] = []
        self._client = anthropic_client
        self._model = model
        if self._client is None:
            self._client = _try_init_anthropic()

    # -- public API --------------------------------------------------------

    def step(self, user_message: str) -> str:
        entry = TimelineEntry(role=ROLE_USER, raw_content=user_message)
        entry.semantic_embedding = self.encoder.encode_entry(entry)
        self._entries.append(entry)

        context = self._retrieve(user_message)
        response = self._generate_response(user_message, context)

        resp_entry = TimelineEntry(role=ROLE_AGENT, raw_content=response)
        resp_entry.semantic_embedding = self.encoder.encode_entry(resp_entry)
        self._entries.append(resp_entry)
        return response

    def decide(
        self, bug_report: str, retrieved_chunks: list[str],
    ) -> DecisionResult:
        """Given a bug report and retrieved text chunks, produce a
        structured investigation decision (with reasoning)."""
        context_text = self._format_baseline_context(retrieved_chunks)
        if self._client:
            return self._decide_claude(bug_report, context_text)
        return self._decide_heuristic(bug_report, context_text)

    # -- context formatting ------------------------------------------------

    @staticmethod
    def _format_baseline_context(chunks: list[str]) -> str:
        if not chunks:
            return "(no relevant past entries)"
        parts = ["## Retrieved Past Entries (from vector memory)"]
        for i, chunk in enumerate(chunks[:5], 1):
            parts.append(f"{i}. {chunk}")
        return "\n".join(parts)

    # -- Claude decision ---------------------------------------------------

    def _decide_claude(
        self, bug_report: str, context_text: str,
    ) -> DecisionResult:
        prompt = (
            f"## Current Bug Report\n{bug_report}\n\n"
            f"{context_text}\n\n"
            f"## Instruction\n{_INVESTIGATION_INSTRUCTION}"
        )
        resp = self._client.messages.create(
            model=self._model,
            max_tokens=400,
            system=(
                "You are a senior SRE debugging a production incident. "
                "Use the retrieved past entries to inform your investigation. "
                "The entries are raw text from previous incidents with no "
                "structure or outcome labels."
            ),
            messages=[{"role": "user", "content": prompt}],
        )
        full_text = resp.content[0].text.strip()
        reasoning, decision = _parse_reasoning_decision(full_text)
        return DecisionResult(
            decision=decision,
            reasoning=reasoning,
            context_used=context_text,
        )

    # -- heuristic fallback ------------------------------------------------

    @staticmethod
    def _decide_heuristic(
        bug_report: str, context_text: str,
    ) -> DecisionResult:
        ctx = context_text.lower()
        bug = bug_report.lower()

        deploy_signals = [
            "after deployment", "after deploy", "since deploy",
            "after release", "after rollout",
        ]
        pool_correction_signals = [
            "not connection pool", "not pool exhaustion",
            "correlated with deployment", "not load",
        ]
        pool_hits = sum(1 for p in _POOL_PATTERNS if p in ctx)
        deploy_hits = sum(1 for s in deploy_signals if s in bug)
        has_pool_correction = any(s in ctx for s in pool_correction_signals)

        if deploy_hits and has_pool_correction:
            decision = (
                "Investigate: recent deployment/config change "
                "-- check migration, schema changes, and rollback"
            )
            reasoning = (
                "Bug report signals deployment timing. Past corrections "
                "warn against the recurring pattern. "
                "Checking deployment artifacts first."
            )
        elif deploy_hits and pool_hits == 0:
            decision = (
                "Investigate: recent deployment/config change "
                "-- verify and rollback"
            )
            reasoning = (
                "Bug report mentions a deployment. No connection pool "
                "signals in retrieved entries. Checking deployment first."
            )
        elif pool_hits:
            decision = "Investigate: connection pool exhaustion under load"
            reasoning = (
                f"Retrieved entries mention connection pool issues "
                f"({pool_hits} references). Suggesting pool exhaustion."
            )
        elif deploy_hits:
            decision = (
                "Investigate: recent deployment/config change "
                "-- verify and rollback"
            )
            reasoning = "Deployment signal in bug report."
        elif "database" in ctx or "db connection" in ctx:
            decision = "Investigate: database connections and query performance"
            reasoning = "Retrieved entries mention database issues."
        elif "network" in ctx or "latency" in ctx or "timeout" in ctx:
            decision = "Investigate: network connectivity and latency"
            reasoning = "Retrieved entries mention network/latency issues."
        else:
            decision = (
                "Investigate: general service health "
                "(no prior pattern match)"
            )
            reasoning = "No matching patterns found in retrieved entries."

        return DecisionResult(
            decision=decision,
            reasoning=reasoning,
            context_used=context_text,
        )

    # -- internals ---------------------------------------------------------

    def _retrieve(self, query: str) -> list[TimelineEntry]:
        if not self._entries:
            return []
        q_emb = self.encoder.encode_text(query)
        scored = []
        for entry in self._entries:
            if entry.semantic_embedding is None:
                continue
            sim = cosine_similarity(q_emb, entry.semantic_embedding)
            scored.append((sim, entry))
        scored.sort(key=lambda x: x[0], reverse=True)
        return [e for _, e in scored[: self.top_k]]

    def _generate_response(
        self, query: str, context: list[TimelineEntry],
    ) -> str:
        snippets = [e.raw_content[:60] for e in context[:3]]
        parts = ["Based on vector memory:"]
        if snippets:
            parts.append(f"  Retrieved: {'; '.join(snippets)}")
        parts.append(f"  Responding to: {query}")
        return "\n".join(parts)


# -- shared helpers --------------------------------------------------------

def _try_init_anthropic() -> Any:
    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        return None
    try:
        import anthropic
        return anthropic.Anthropic(api_key=api_key)
    except Exception:
        return None


def _parse_reasoning_decision(text: str) -> tuple[str, str]:
    """Extract REASONING and DECISION sections from Claude's response."""
    reasoning = text
    decision = text

    lines = text.split("\n")
    r_parts: list[str] = []
    d_parts: list[str] = []
    section = None

    for line in lines:
        upper = line.strip().upper()
        if upper.startswith("REASONING:"):
            section = "r"
            r_parts.append(line.strip()[len("REASONING:"):].strip())
        elif upper.startswith("DECISION:"):
            section = "d"
            d_parts.append(line.strip()[len("DECISION:"):].strip())
        elif section == "r":
            r_parts.append(line.strip())
        elif section == "d":
            d_parts.append(line.strip())

    if r_parts:
        reasoning = " ".join(p for p in r_parts if p)
    if d_parts:
        decision = " ".join(p for p in d_parts if p)

    return reasoning, decision

"""Episodic-memory agent.

Demonstrates the full loop: perceive -> recall -> decide -> reinforce.

When ANTHROPIC_API_KEY is set, investigation decisions are made by Claude
using structured episodic context (episodes with outcomes/corrections,
generalized facts with support counts).  Falls back to keyword heuristics
when no key is available.
"""

from __future__ import annotations

import os
from typing import Any

from memory.encoder import Encoder
from memory.episodes import EpisodeBoundaryDetector
from memory.facts import FactExtractor
from memory.graph import EpisodeGraph
from memory.models import (
    OUTCOME_FAILURE,
    OUTCOME_SUCCESS,
    DecisionResult,
    RetrievalResult,
    ROLE_AGENT,
    ROLE_USER,
)
from memory.reinforcer import Reinforcer
from memory.retriever import Retriever
from memory.timeline import Timeline


_POOL_PATTERNS = [
    "connection pool", "pool exhaust", "pool saturat",
    "pool starvation", "pool contention",
]

_INVESTIGATION_INSTRUCTION = (
    "Based on your past experience shown above, what should be investigated "
    "FIRST? Be specific. Explain your reasoning citing the past episodes "
    "and facts.\n\n"
    "Respond in this exact format:\n"
    "REASONING: <2-4 sentences analyzing the pattern from past episodes>\n"
    "DECISION: Investigate: <one specific sentence>"
)


class EpisodicAgent:
    """Agent backed by the full episodic memory stack."""

    def __init__(
        self,
        encoder: Encoder | None = None,
        anthropic_client: Any = None,
        model: str = "claude-sonnet-4-20250514",
    ) -> None:
        self.encoder = encoder or Encoder()
        self.timeline = Timeline()
        self.episode_builder = EpisodeBoundaryDetector(self.encoder)
        self.graph = EpisodeGraph()
        self.fact_extractor = FactExtractor(self.encoder)
        self.retriever = Retriever(
            encoder=self.encoder,
            graph=self.graph,
            fact_extractor=self.fact_extractor,
        )
        self.reinforcer = Reinforcer(
            encoder=self.encoder,
            graph=self.graph,
            fact_extractor=self.fact_extractor,
        )
        self._client = anthropic_client
        self._model = model
        if self._client is None:
            self._client = _try_init_anthropic()

    # -- public API --------------------------------------------------------

    def step(self, user_message: str) -> str:
        """Process one user turn: record -> retrieve -> respond -> reinforce."""
        self.timeline.append(ROLE_USER, user_message)
        self._rebuild_episodes()

        context = self.retriever.retrieve(user_message)
        response = self._generate_response(user_message, context)
        self.timeline.append(ROLE_AGENT, response)

        outcome, details = self._evaluate_outcome(user_message, response)
        self.reinforcer.reinforce(context, outcome, outcome_details=details)
        return response

    def decide(
        self, bug_report: str, result: RetrievalResult,
    ) -> DecisionResult:
        """Given a bug report and retrieved episodic context, produce a
        structured investigation decision (with reasoning)."""
        context_text = self._format_episodic_context(result)
        if self._client:
            return self._decide_claude(bug_report, context_text)
        return self._decide_heuristic(bug_report, context_text)

    # -- context formatting ------------------------------------------------

    @staticmethod
    def _format_episodic_context(result: RetrievalResult) -> str:
        parts: list[str] = []

        if result.episodes:
            parts.append("## Past Episodes (from episodic memory)")
            for i, ep in enumerate(result.episodes[:5], 1):
                outcome = ep.outcome.upper() if ep.outcome else "UNKNOWN"
                parts.append(f"Episode {i}: {ep.summary_text}")
                parts.append(f"  [OUTCOME: {outcome}]")
                if ep.assumptions:
                    parts.append(
                        f"  Initial assumption: {'; '.join(ep.assumptions)}"
                    )
                if ep.corrections:
                    parts.append(
                        f"  Correction: {'; '.join(ep.corrections)}"
                    )
                if ep.outcome_details:
                    parts.append(f"  Root cause: {ep.outcome_details}")
                parts.append("")

        if result.facts:
            parts.append("## Generalized Facts (learned from multiple episodes)")
            for f in result.facts[:5]:
                parts.append(
                    f"- {f.fact_text}  "
                    f"(supported by {f.support_count} episodes, "
                    f"contradicted by {f.contradiction_count})"
                )
            parts.append("")

        if result.causal_narrative:
            parts.append(f"## Causal Chain\n{result.causal_narrative}\n")

        return "\n".join(parts) or "(no relevant past experience)"

    # -- Claude decision ---------------------------------------------------

    def _decide_claude(
        self, bug_report: str, context_text: str,
    ) -> DecisionResult:
        prompt = (
            f"## Current Bug Report\n{bug_report}\n\n"
            f"{context_text}\n"
            f"## Instruction\n{_INVESTIGATION_INSTRUCTION}"
        )
        resp = self._client.messages.create(
            model=self._model,
            max_tokens=400,
            system=(
                "You are a senior SRE debugging a production incident. "
                "Use your episodic memory of past incidents to inform your "
                "investigation. Be specific about which past episodes "
                "support your hypothesis."
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
                "explicitly warn against the recurring pattern. "
                "Checking deployment artifacts and migrations first."
            )
        elif deploy_hits and pool_hits == 0:
            decision = (
                "Investigate: recent deployment/config change "
                "-- verify and rollback"
            )
            reasoning = (
                "Bug report mentions a deployment. No connection pool "
                "signals in past experience. Prioritizing deployment check."
            )
        elif pool_hits:
            decision = "Investigate: connection pool exhaustion under load"
            reasoning = (
                f"Past episodes mention connection pool issues "
                f"({pool_hits} references). Pattern suggests pool "
                f"exhaustion as recurring root cause under load."
            )
        elif deploy_hits:
            decision = (
                "Investigate: recent deployment/config change "
                "-- verify and rollback"
            )
            reasoning = "Deployment signal in bug report."
        elif "database" in ctx or "db connection" in ctx:
            decision = "Investigate: database connections and query performance"
            reasoning = "Past context mentions database issues."
        elif "network" in ctx or "latency" in ctx or "timeout" in ctx:
            decision = "Investigate: network connectivity and latency"
            reasoning = "Past context mentions network/latency issues."
        else:
            decision = (
                "Investigate: general service health "
                "(no prior pattern match)"
            )
            reasoning = "No matching patterns found in past experience."

        return DecisionResult(
            decision=decision,
            reasoning=reasoning,
            context_used=context_text,
        )

    # -- internals ---------------------------------------------------------

    def _rebuild_episodes(self) -> None:
        episodes = self.episode_builder.build_episodes(self.timeline.entries)
        for ep in episodes:
            if ep.episode_id not in self.graph.episode_ids:
                self.graph.add_episode(ep)
                self.fact_extractor.extract(ep)
        self.graph.auto_link(episodes)

    def _generate_response(self, query: str, context: RetrievalResult) -> str:
        episode_summaries = [ep.summary_text for ep in context.episodes[:3]]
        fact_statements = [f.fact_text for f in context.facts[:2]]
        parts = ["Based on memory:"]
        if episode_summaries:
            parts.append(f"  Episodes: {'; '.join(episode_summaries)}")
        if fact_statements:
            parts.append(f"  Facts: {'; '.join(fact_statements)}")
        parts.append(f"  Responding to: {query}")
        return "\n".join(parts)

    @staticmethod
    def _evaluate_outcome(query: str, response: str) -> tuple[str, str]:
        if "error" in response.lower():
            return OUTCOME_FAILURE, "Response contained an error"
        return OUTCOME_SUCCESS, "Response generated successfully"


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
    """Extract REASONING and DECISION sections from Claude's response.
    Returns (reasoning, decision).  Falls back to full text if parsing fails.
    """
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

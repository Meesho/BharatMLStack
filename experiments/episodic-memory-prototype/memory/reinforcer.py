"""Outcome → memory feedback loop.

After every agent decision with a known outcome:
  1. Create a new outcome Episode capturing what happened.
  2. Link it to the retrieved episodes via LINK_LEARNED_FROM.
  3. Call Claude to check whether each relevant fact is supported or
     contradicted by the outcome.
  4. Update support_count / contradiction_count on each fact.

Falls back to a simple heuristic (positive → support, negative →
contradict) when Claude is unavailable.
"""

from __future__ import annotations

import logging
import os
from typing import Any

from memory.encoder import Encoder
from memory.facts import FactExtractor
from memory.graph import EpisodeGraph
from memory.models import (
    Episode,
    EpisodeLink,
    GeneralizedFact,
    LINK_LEARNED_FROM,
    LINK_SOURCE_INFERRED,
    OUTCOME_FAILURE,
    OUTCOME_SUCCESS,
    OUTCOME_UNKNOWN,
    RetrievalResult,
    _now,
)

log = logging.getLogger(__name__)

_VERDICT_SUPPORTS = "supports"
_VERDICT_CONTRADICTS = "contradicts"
_VERDICT_NEUTRAL = "neutral"


class Reinforcer:
    """Feeds outcome signals back into episodic memory."""

    def __init__(
        self,
        encoder: Encoder,
        graph: EpisodeGraph,
        fact_extractor: FactExtractor,
        decay: float = 0.9,
        anthropic_client: Any = None,
        model: str = "claude-sonnet-4-20250514",
    ) -> None:
        self.encoder = encoder
        self.graph = graph
        self.fact_extractor = fact_extractor
        self.decay = decay
        self._model = model
        self._anthropic = anthropic_client or _try_init_anthropic()

    def reinforce(
        self,
        result: RetrievalResult,
        outcome: str,
        outcome_details: str = "",
    ) -> Episode | None:
        """Full feedback loop: episode → links → fact-check → count update."""
        if not result.episodes:
            return None

        outcome_ep = self._create_outcome_episode(result, outcome, outcome_details)
        self.graph.add_episode(outcome_ep)
        self._link_to_retrieved(outcome_ep, result)

        self._update_episode_scores(result, outcome)
        self._check_and_update_facts(result.facts, outcome, outcome_details)

        return outcome_ep

    # -- outcome episode -------------------------------------------------------

    def _create_outcome_episode(
        self,
        result: RetrievalResult,
        outcome: str,
        outcome_details: str,
    ) -> Episode:
        summary = f"Outcome: {outcome}"
        if outcome_details:
            summary += f" — {outcome_details}"

        ref_state = result.episodes[0].state if result.episodes else ""

        return Episode(
            summary_text=summary,
            summary_embedding=self.encoder.encode_text(summary),
            outcome=outcome,
            outcome_details=outcome_details,
            state=ref_state,
            start_time=_now(),
            end_time=_now(),
        )

    # -- linking ---------------------------------------------------------------

    def _link_to_retrieved(
        self, outcome_ep: Episode, result: RetrievalResult
    ) -> None:
        for ep in result.episodes:
            link = EpisodeLink(
                source_episode_id=outcome_ep.episode_id,
                target_episode_id=ep.episode_id,
                link_type=LINK_LEARNED_FROM,
                source=LINK_SOURCE_INFERRED,
                evidence=f"Outcome '{outcome_ep.outcome}' after retrieving this episode",
            )
            self.graph.add_link(link)

    # -- episode score update --------------------------------------------------

    def _update_episode_scores(
        self, result: RetrievalResult, outcome: str
    ) -> None:
        score = self._outcome_to_score(outcome)
        for i, episode in enumerate(result.episodes):
            weight = self.decay ** i
            ep = self.graph.get_episode(episode.episode_id)
            if ep is not None:
                ep.outcome_score = (
                    ep.outcome_score * 0.7 + score * weight * 0.3
                )

    # -- fact checking ---------------------------------------------------------

    def _check_and_update_facts(
        self,
        facts: list[GeneralizedFact],
        outcome: str,
        outcome_details: str,
    ) -> None:
        for fact in facts:
            stored = self._find_stored_fact(fact.fact_id)
            if stored is None:
                continue

            verdict = self._check_fact(stored.fact_text, outcome, outcome_details)

            if verdict == _VERDICT_SUPPORTS:
                stored.support_count += 1
                stored.last_updated = _now()
            elif verdict == _VERDICT_CONTRADICTS:
                stored.contradiction_count += 1
                stored.last_updated = _now()

    def _check_fact(
        self, fact_text: str, outcome: str, outcome_details: str
    ) -> str:
        if self._anthropic is not None:
            try:
                return self._call_claude_fact_check(
                    fact_text, outcome, outcome_details
                )
            except Exception:
                log.debug("Claude fact-check failed, using heuristic", exc_info=True)

        return self._heuristic_verdict(outcome)

    def _call_claude_fact_check(
        self, fact_text: str, outcome: str, outcome_details: str
    ) -> str:
        outcome_desc = outcome
        if outcome_details:
            outcome_desc += f" — {outcome_details}"

        response = self._anthropic.messages.create(
            model=self._model,
            max_tokens=20,
            messages=[{
                "role": "user",
                "content": (
                    "An agent made a decision and got this outcome:\n"
                    f"Outcome: {outcome_desc}\n\n"
                    "Does this outcome support or contradict the following "
                    "learned fact?\n"
                    f'Fact: "{fact_text}"\n\n'
                    "Answer with exactly one word: SUPPORTS or CONTRADICTS or NEUTRAL"
                ),
            }],
        )
        raw = response.content[0].text.strip().lower()
        if "support" in raw:
            return _VERDICT_SUPPORTS
        if "contradict" in raw:
            return _VERDICT_CONTRADICTS
        return _VERDICT_NEUTRAL

    @staticmethod
    def _heuristic_verdict(outcome: str) -> str:
        if outcome in (OUTCOME_SUCCESS,):
            return _VERDICT_SUPPORTS
        if outcome in (OUTCOME_FAILURE,):
            return _VERDICT_CONTRADICTS
        return _VERDICT_NEUTRAL

    @staticmethod
    def _outcome_to_score(outcome: str) -> float:
        if outcome == OUTCOME_SUCCESS:
            return 1.0
        if outcome == OUTCOME_FAILURE:
            return -1.0
        return 0.0

    def _find_stored_fact(self, fact_id: str) -> GeneralizedFact | None:
        for f in self.fact_extractor.facts:
            if f.fact_id == fact_id:
                return f
        return None


def _try_init_anthropic() -> Any:
    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        return None
    try:
        import anthropic
        return anthropic.Anthropic(api_key=api_key)
    except Exception:
        return None

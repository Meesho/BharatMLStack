"""Generalized fact extraction.

Single-episode calls accumulate in a buffer.  When 3+ episodes have been
seen, Claude is called to find cross-episode patterns.  Falls back to
embedding-merge when no API key / package is available.
"""

from __future__ import annotations

import logging
import os
from typing import Any

from memory.encoder import Encoder
from memory.models import Episode, GeneralizedFact, cosine_similarity, _now

log = logging.getLogger(__name__)


class FactExtractor:
    """Extracts and maintains a corpus of generalized facts."""

    def __init__(
        self,
        encoder: Encoder,
        merge_threshold: float = 0.80,
        min_episodes_for_extraction: int = 3,
        anthropic_client: Any = None,
        model: str = "claude-sonnet-4-20250514",
    ) -> None:
        self.encoder = encoder
        self.merge_threshold = merge_threshold
        self.min_episodes_for_extraction = min_episodes_for_extraction
        self._model = model
        self._anthropic = anthropic_client or _try_init_anthropic()
        self._facts: list[GeneralizedFact] = []
        self._episode_buffer: list[Episode] = []

    @property
    def facts(self) -> list[GeneralizedFact]:
        return list(self._facts)

    # -- single-episode entry point (backward compat) --------------------------

    def extract(self, episode: Episode) -> list[GeneralizedFact]:
        """Accumulate one episode.  Triggers pattern extraction at threshold."""
        self._merge_or_add_basic(episode)
        self._episode_buffer.append(episode)

        if len(self._episode_buffer) >= self.min_episodes_for_extraction:
            self._extract_patterns(list(self._episode_buffer))
            self._episode_buffer.clear()

        return self._facts

    # -- explicit batch entry point --------------------------------------------

    def extract_patterns(self, episodes: list[Episode]) -> list[GeneralizedFact]:
        """Call Claude to find patterns across 3+ episodes."""
        if len(episodes) < self.min_episodes_for_extraction:
            for ep in episodes:
                self._merge_or_add_basic(ep)
            return self._facts

        self._extract_patterns(episodes)
        return self._facts

    # -- query -----------------------------------------------------------------

    def query(self, text: str, top_k: int = 5) -> list[GeneralizedFact]:
        if not self._facts:
            return []
        q_emb = self.encoder.encode_text(text)
        scored = []
        for f in self._facts:
            f_emb = self._fact_embedding(f)
            scored.append((cosine_similarity(q_emb, f_emb), f))
        scored.sort(key=lambda x: x[0], reverse=True)
        return [f for _, f in scored[:top_k]]

    # -- internals -------------------------------------------------------------

    def _extract_patterns(self, episodes: list[Episode]) -> None:
        summaries = [
            ep.summary_text for ep in episodes if ep.summary_text
        ]
        if not summaries:
            return

        episode_ids = [ep.episode_id for ep in episodes]

        if self._anthropic is not None:
            try:
                patterns = self._call_claude_patterns(summaries)
                for text in patterns:
                    self._merge_or_add(text.strip(), episode_ids)
                return
            except Exception:
                log.debug("Claude pattern extraction failed, using fallback", exc_info=True)

        for ep in episodes:
            self._merge_or_add_basic(ep)

    def _call_claude_patterns(self, summaries: list[str]) -> list[str]:
        numbered = "\n".join(f"{i+1}. {s}" for i, s in enumerate(summaries))
        response = self._anthropic.messages.create(
            model=self._model,
            max_tokens=300,
            messages=[{
                "role": "user",
                "content": (
                    "Given these episode summaries from an agent's experience:\n\n"
                    f"{numbered}\n\n"
                    "Identify 1-3 generalized patterns, heuristics, or lessons "
                    "that emerge. Each should be a single declarative sentence "
                    "that could guide future decisions.\n\n"
                    "Return ONLY the patterns, one per line, no numbering or bullets."
                ),
            }],
        )
        raw = response.content[0].text.strip()
        return [line for line in raw.splitlines() if line.strip()]

    def _merge_or_add_basic(self, episode: Episode) -> None:
        """Merge a single episode's summary into facts (no LLM)."""
        if not episode.summary_text:
            return
        self._merge_or_add(episode.summary_text, [episode.episode_id])

    def _merge_or_add(self, fact_text: str, episode_ids: list[str]) -> None:
        embedding = self.encoder.encode_text(fact_text)

        for existing in self._facts:
            existing_emb = self._fact_embedding(existing)
            if cosine_similarity(embedding, existing_emb) >= self.merge_threshold:
                existing.support_count += 1
                existing.last_updated = _now()
                for eid in episode_ids:
                    if eid not in existing.supporting_episodes:
                        existing.supporting_episodes.append(eid)
                return

        self._facts.append(GeneralizedFact(
            fact_text=fact_text,
            supporting_episodes=list(episode_ids),
            fact_embedding=embedding,
        ))

    def _fact_embedding(self, fact: GeneralizedFact) -> list[float]:
        if fact.fact_embedding is not None:
            return fact.fact_embedding
        emb = self.encoder.encode_text(fact.fact_text)
        fact.fact_embedding = emb
        return emb


def _try_init_anthropic() -> Any:
    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        return None
    try:
        import anthropic
        return anthropic.Anthropic(api_key=api_key)
    except Exception:
        return None

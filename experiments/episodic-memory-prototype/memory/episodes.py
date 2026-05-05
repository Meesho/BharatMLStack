"""Episode boundary detection and creation.

Boundary signal is a weighted combination of two features:
  - state change  (weight 0.6): binary — did the state_label flip?
  - semantic shift (weight 0.4): 1 − cosine_sim between consecutive entries.

When the combined score crosses the threshold (default 0.5), a new episode
starts.  Episode summaries are generated via the Claude API when available,
with a plain-text fallback for tests / offline use.
"""

from __future__ import annotations

import logging
import os
from typing import Any

from memory.encoder import Encoder
from memory.models import (
    Episode,
    OUTCOME_UNKNOWN,
    TimelineEntry,
    cosine_similarity,
)

log = logging.getLogger(__name__)


class EpisodeBoundaryDetector:
    """Detects episode boundaries and builds Episode objects."""

    def __init__(
        self,
        encoder: Encoder,
        threshold: float = 0.5,
        state_change_weight: float = 0.6,
        semantic_shift_weight: float = 0.4,
        anthropic_client: Any = None,
        model: str = "claude-sonnet-4-20250514",
    ) -> None:
        self.encoder = encoder
        self.threshold = threshold
        self.state_change_weight = state_change_weight
        self.semantic_shift_weight = semantic_shift_weight
        self._model = model
        self._anthropic = anthropic_client or self._try_init_anthropic()

    # -- public API ------------------------------------------------------------

    def build_episodes(self, entries: list[TimelineEntry]) -> list[Episode]:
        """Segment entries into episodes and return them."""
        if not entries:
            return []

        self._ensure_embeddings(entries)

        boundaries = self.detect_boundaries(entries)
        segments = self._split_at_boundaries(entries, boundaries)
        return [self._segment_to_episode(seg) for seg in segments]

    def detect_boundaries(self, entries: list[TimelineEntry]) -> list[int]:
        """Return indices (1-based into *entries*) where a new episode starts.

        Index *i* in the result means a boundary exists *before* entries[i].
        """
        self._ensure_embeddings(entries)

        boundaries: list[int] = []
        for i in range(1, len(entries)):
            score = self._boundary_score(entries[i - 1], entries[i])
            if score >= self.threshold:
                boundaries.append(i)
        return boundaries

    # -- boundary scoring ------------------------------------------------------

    def _boundary_score(self, prev: TimelineEntry, curr: TimelineEntry) -> float:
        state_change = 1.0 if (
            prev.state_label != curr.state_label
            and prev.state_label != ""
            and curr.state_label != ""
        ) else 0.0

        if prev.semantic_embedding and curr.semantic_embedding:
            sim = cosine_similarity(prev.semantic_embedding, curr.semantic_embedding)
            semantic_shift = 1.0 - max(sim, 0.0)
        else:
            semantic_shift = 0.0

        return (
            self.state_change_weight * state_change
            + self.semantic_shift_weight * semantic_shift
        )

    # -- segment → episode -----------------------------------------------------

    @staticmethod
    def _split_at_boundaries(
        entries: list[TimelineEntry], boundaries: list[int]
    ) -> list[list[TimelineEntry]]:
        segments: list[list[TimelineEntry]] = []
        prev = 0
        for b in boundaries:
            if prev < b:
                segments.append(entries[prev:b])
            prev = b
        if prev < len(entries):
            segments.append(entries[prev:])
        return segments

    def _segment_to_episode(self, segment: list[TimelineEntry]) -> Episode:
        summary = self._summarize_entries(segment)
        return Episode(
            start_time=segment[0].timestamp,
            end_time=segment[-1].timestamp,
            timeline_start=segment[0].sequence_id,
            timeline_end=segment[-1].sequence_id,
            entry_ids=[e.entry_id for e in segment],
            entry_count=len(segment),
            state=segment[0].state_label,
            summary_text=summary,
            summary_embedding=self.encoder.encode_text(summary),
            outcome=OUTCOME_UNKNOWN,
        )

    # -- summary generation ----------------------------------------------------

    def _summarize_entries(self, entries: list[TimelineEntry]) -> str:
        content = "\n".join(
            f"[{e.role}] {e.raw_content}" for e in entries
        )

        if self._anthropic is not None:
            try:
                return self._call_claude(content)
            except Exception:
                log.debug("Claude API call failed, using fallback summary", exc_info=True)

        return self._fallback_summary(entries)

    def _call_claude(self, content: str) -> str:
        response = self._anthropic.messages.create(
            model=self._model,
            max_tokens=150,
            messages=[
                {
                    "role": "user",
                    "content": (
                        "Summarize this agent interaction in exactly 2 sentences. "
                        "Focus on what was discussed and what was decided or resolved.\n\n"
                        f"{content}"
                    ),
                }
            ],
        )
        return response.content[0].text.strip()

    @staticmethod
    def _fallback_summary(entries: list[TimelineEntry]) -> str:
        first = entries[0].raw_content[:80]
        return f"{len(entries)} entries: {first}…"

    # -- helpers ---------------------------------------------------------------

    def _ensure_embeddings(self, entries: list[TimelineEntry]) -> None:
        for e in entries:
            if e.semantic_embedding is None:
                e.semantic_embedding = self.encoder.encode_entry(e)

    @staticmethod
    def _try_init_anthropic() -> Any:
        api_key = os.environ.get("ANTHROPIC_API_KEY")
        if not api_key:
            return None
        try:
            import anthropic
            return anthropic.Anthropic(api_key=api_key)
        except Exception:
            return None

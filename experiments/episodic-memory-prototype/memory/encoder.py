"""Embedding and state encoding.

Three kinds of encoding:
  1. Semantic — sentence-transformers (all-MiniLM-L6-v2) for content meaning.
     Falls back to a deterministic hash projection when the model is
     unavailable (keeps tests fast and dependency-free).
  2. State   — fixed dictionary that maps state-label strings to
     random-but-consistent vectors (seeded from the label hash).
  3. Time    — exponential decay scalar: exp(-λ · Δhours).  Not a full
     positional encoding; just a recency weight between 0 and 1.
"""

from __future__ import annotations

import hashlib
import math
from datetime import datetime, timezone
from typing import Any

import numpy as np

from memory.models import TimelineEntry


class Encoder:
    """Produces semantic embeddings, state embeddings, and time-decay scalars."""

    def __init__(
        self,
        model_name: str = "all-MiniLM-L6-v2",
        dim: int = 384,
        time_half_life_hours: float = 24.0,
    ) -> None:
        self.dim = dim
        self._model_name = model_name
        self._model: Any = None
        self._model_load_attempted = False
        self._state_cache: dict[str, list[float]] = {}
        self._time_half_life_hours = time_half_life_hours
        self._time_lambda = math.log(2) / max(time_half_life_hours, 1e-9)

    # -- model loading ---------------------------------------------------------

    def _load_model(self) -> Any:
        if not self._model_load_attempted:
            self._model_load_attempted = True
            try:
                from sentence_transformers import SentenceTransformer
                self._model = SentenceTransformer(self._model_name)
            except Exception:
                self._model = None
        return self._model

    # -- semantic embeddings ---------------------------------------------------

    def encode_text(self, text: str) -> list[float]:
        """Embed arbitrary text via sentence-transformers (or hash fallback)."""
        model = self._load_model()
        if model is not None:
            vec = model.encode(text, normalize_embeddings=True)
            return vec.tolist()
        return self._deterministic_hash_embed(text)

    def encode_entry(self, entry: TimelineEntry) -> list[float]:
        """Semantic embedding for a timeline entry's content."""
        return self.encode_text(f"[{entry.role}] {entry.raw_content}")

    # -- state embeddings ------------------------------------------------------

    def encode_state(self, state_label: str) -> list[float]:
        """Deterministic random vector for a state label.

        Same label always produces the same vector; different labels produce
        near-orthogonal vectors.  Cached after first computation.
        """
        if not state_label:
            state_label = "__empty__"

        if state_label in self._state_cache:
            return self._state_cache[state_label]

        vec = self._deterministic_hash_embed(f"__state__:{state_label}")
        self._state_cache[state_label] = vec
        return vec

    # -- time decay ------------------------------------------------------------

    def time_decay(
        self,
        entry_ts: datetime,
        reference: datetime | None = None,
    ) -> float:
        """Exponential decay scalar in (0, 1].

        Returns 1.0 for *reference* itself and decays toward 0 as *entry_ts*
        gets older.  Half-life is configurable at init (default 24 h).
        """
        ref = reference or datetime.now(timezone.utc)
        delta_hours = max((ref - entry_ts).total_seconds() / 3600.0, 0.0)
        return math.exp(-self._time_lambda * delta_hours)

    # -- hash fallback (no ML model needed) ------------------------------------

    def _deterministic_hash_embed(self, text: str) -> list[float]:
        """Reproducible pseudo-embedding derived from SHA-256."""
        digest = hashlib.sha256(text.encode()).digest()
        rng = np.random.Generator(np.random.PCG64(int.from_bytes(digest[:8], "big")))
        vec = rng.standard_normal(self.dim).astype(np.float32)
        vec /= np.linalg.norm(vec) + 1e-9
        return vec.tolist()

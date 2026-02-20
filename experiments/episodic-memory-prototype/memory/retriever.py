"""Query-time retrieval.

Retrieval strategy:
  1. Score all episodes by cosine similarity to the query embedding.
  2. Boost episodes that match the current state (×1.5).
  3. Boost episodes whose outcome is FAILURE (×1.25 — failures teach more).
  4. Optionally traverse graph 1 hop to pull in connected episodes.
  5. Select top-K with MMR: penalize candidates similar to already-selected.
  6. Ensure at least one SUCCESS (counter-example) episode if any exists.
  7. Fetch generalized facts, preferring ones applicable to the current state.
  8. Return a RetrievalResult with episodes, facts, scores, and a narrative.
"""

from __future__ import annotations

from memory.encoder import Encoder
from memory.facts import FactExtractor
from memory.graph import EpisodeGraph
from memory.models import (
    Episode,
    OUTCOME_FAILURE,
    OUTCOME_SUCCESS,
    RetrievalResult,
    cosine_similarity,
)


class Retriever:
    """Assembles a ranked context window from episodic memory."""

    def __init__(
        self,
        encoder: Encoder,
        graph: EpisodeGraph,
        fact_extractor: FactExtractor,
        top_k_episodes: int = 5,
        top_k_facts: int = 3,
        graph_depth: int = 1,
        failure_boost: float = 1.25,
        state_match_boost: float = 1.5,
    ) -> None:
        self.encoder = encoder
        self.graph = graph
        self.fact_extractor = fact_extractor
        self.top_k_episodes = top_k_episodes
        self.top_k_facts = top_k_facts
        self.graph_depth = graph_depth
        self.failure_boost = failure_boost
        self.state_match_boost = state_match_boost

    def retrieve(
        self,
        query: str,
        current_state: str = "",
        traverse: bool = True,
    ) -> RetrievalResult:
        q_emb = self.encoder.encode_text(query)

        scored = self._score_all_episodes(q_emb, current_state)

        if traverse and self.graph_depth > 0:
            scored = self._expand_via_graph(scored, q_emb, current_state)

        top = self._select_with_mmr(scored, self.top_k_episodes)
        episodes, scores = self._ensure_counter_example(scored, top)
        facts = self._fetch_facts(query, current_state)
        narrative = self._build_narrative(episodes)

        return RetrievalResult(
            episodes=episodes,
            facts=facts,
            scores=scores,
            causal_narrative=narrative,
        )

    # -- scoring ---------------------------------------------------------------

    def _score_all_episodes(
        self, q_emb: list[float], current_state: str
    ) -> list[tuple[float, Episode]]:
        results: list[tuple[float, Episode]] = []
        for eid in self.graph.episode_ids:
            ep = self.graph.get_episode(eid)
            if ep is None:
                continue
            score = self._episode_score(q_emb, ep, current_state)
            results.append((score, ep))
        results.sort(key=lambda x: x[0], reverse=True)
        return results

    def _episode_score(
        self, q_emb: list[float], episode: Episode, current_state: str = ""
    ) -> float:
        if not episode.summary_embedding:
            return 0.0
        sim = cosine_similarity(q_emb, episode.summary_embedding)
        if episode.outcome == OUTCOME_FAILURE:
            sim *= self.failure_boost
        if current_state and episode.state == current_state:
            sim *= self.state_match_boost
        return sim

    # -- graph expansion -------------------------------------------------------

    def _expand_via_graph(
        self,
        scored: list[tuple[float, Episode]],
        q_emb: list[float],
        current_state: str,
    ) -> list[tuple[float, Episode]]:
        seen: dict[str, tuple[float, Episode]] = {
            ep.episode_id: (s, ep) for s, ep in scored
        }
        for _, ep in list(scored[: self.top_k_episodes]):
            for neighbor in self.graph.traverse(
                ep.episode_id, depth=self.graph_depth
            ):
                if neighbor.episode_id in seen:
                    continue
                n_score = (
                    self._episode_score(q_emb, neighbor, current_state) * 0.8
                )
                seen[neighbor.episode_id] = (n_score, neighbor)

        result = list(seen.values())
        result.sort(key=lambda x: x[0], reverse=True)
        return result

    # -- MMR and counter-example ------------------------------------------------

    def _select_with_mmr(
        self,
        scored: list[tuple[float, Episode]],
        top_k: int,
        mmr_penalty: float = 0.5,
    ) -> list[tuple[float, Episode]]:
        """Select top_k episodes using maximal marginal relevance.
        For each slot after the first, penalize score by similarity to already
        selected: new_score = score * (1 - mmr_penalty * max_sim_to_selected).
        """
        if not scored or top_k <= 0:
            return []
        selected: list[tuple[float, Episode]] = []
        remaining = list(scored)

        # Top-1: take highest-scoring
        remaining.sort(key=lambda x: x[0], reverse=True)
        best = remaining.pop(0)
        selected.append(best)

        for _ in range(top_k - 1):
            if not remaining:
                break
            best_idx = -1
            best_mmr_score = -1.0
            for i, (score, ep) in enumerate(remaining):
                if not ep.summary_embedding:
                    mmr_score = score
                else:
                    max_sim = 0.0
                    for _, sel_ep in selected:
                        if sel_ep.summary_embedding:
                            sim = cosine_similarity(
                                ep.summary_embedding,
                                sel_ep.summary_embedding,
                            )
                            max_sim = max(max_sim, sim)
                    mmr_score = score * (1.0 - mmr_penalty * max_sim)
                if mmr_score > best_mmr_score:
                    best_mmr_score = mmr_score
                    best_idx = i
            if best_idx < 0:
                break
            chosen = remaining.pop(best_idx)
            selected.append(chosen)

        return selected

    def _ensure_counter_example(
        self,
        scored: list[tuple[float, Episode]],
        selected: list[tuple[float, Episode]],
    ) -> tuple[list[Episode], list[float]]:
        """If any episode has outcome=SUCCESS and none is in selected,
        add one as counter-example. Return (episodes, scores) in order.
        """
        selected_eps = [ep for _, ep in selected]
        selected_ids = {ep.episode_id for ep in selected_eps}
        has_success = any(ep.outcome == OUTCOME_SUCCESS for ep in selected_eps)

        if has_success:
            episodes = selected_eps
            scores = [s for s, _ in selected]
            return (episodes, scores)

        # Find best SUCCESS episode not already selected (by original score)
        success_candidates = [
            (s, ep) for s, ep in scored
            if ep.outcome == OUTCOME_SUCCESS and ep.episode_id not in selected_ids
        ]
        if not success_candidates:
            episodes = selected_eps
            scores = [s for s, _ in selected]
            return (episodes, scores)

        success_candidates.sort(key=lambda x: x[0], reverse=True)
        add_score, add_ep = success_candidates[0]
        episodes = selected_eps + [add_ep]
        scores = [s for s, _ in selected] + [add_score]
        return (episodes, scores)

    # -- facts -----------------------------------------------------------------

    def _fetch_facts(self, query: str, state: str) -> list:
        all_facts = self.fact_extractor.query(query, top_k=self.top_k_facts * 2)
        if not state:
            return all_facts[: self.top_k_facts]

        state_matched = [
            f for f in all_facts
            if not f.applicable_states or state in f.applicable_states
        ]
        return state_matched[: self.top_k_facts] or all_facts[: self.top_k_facts]

    # -- narrative -------------------------------------------------------------

    @staticmethod
    def _build_narrative(episodes: list[Episode]) -> str:
        if not episodes:
            return ""
        parts: list[str] = []
        for ep in episodes:
            label = ep.summary_text[:60] if ep.summary_text else ep.episode_id
            outcome_tag = f", outcome={ep.outcome}" if ep.outcome else ""
            state_tag = f" [{ep.state}]" if ep.state else ""
            parts.append(f"{label}{state_tag}{outcome_tag}")
        return " → ".join(parts)

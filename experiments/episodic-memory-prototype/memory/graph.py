"""Episode graph — links and traversal.

Episodes stored as a dict, links as a list.  No Neo4j.

Auto-link inference produces two relationship types:
  - CONTINUATION: same state, close in time (chronologically adjacent).
  - RETRY_OF:     similar embedding + same state + the earlier episode failed.
"""

from __future__ import annotations

from collections import defaultdict
from typing import Sequence

from memory.models import (
    Episode,
    EpisodeLink,
    LINK_CONTINUATION,
    LINK_RETRY_OF,
    LINK_SOURCE_INFERRED,
    OUTCOME_FAILURE,
    cosine_similarity,
)


class EpisodeGraph:
    """Directed graph over episodes.  Dict + list, nothing fancy."""

    def __init__(self) -> None:
        self._episodes: dict[str, Episode] = {}
        self._links: list[EpisodeLink] = []
        self._out_index: dict[str, list[int]] = defaultdict(list)

    # -- mutations -------------------------------------------------------------

    def add_episode(self, episode: Episode) -> None:
        self._episodes[episode.episode_id] = episode

    def add_link(self, link: EpisodeLink) -> None:
        idx = len(self._links)
        self._links.append(link)
        self._out_index[link.source_episode_id].append(idx)

    def auto_link(
        self,
        episodes: list[Episode],
        sim_threshold: float = 0.55,
        max_gap_seconds: float = 3600,
    ) -> list[EpisodeLink]:
        """Infer CONTINUATION and RETRY_OF links from a batch of episodes."""
        new_links: list[EpisodeLink] = []

        for ep in episodes:
            self.add_episode(ep)

        by_time = sorted(episodes, key=lambda e: e.start_time)

        for prev, curr in zip(by_time[:-1], by_time[1:]):
            if not prev.state or not curr.state:
                continue
            if prev.state != curr.state:
                continue
            gap = (curr.start_time - prev.end_time).total_seconds()
            if gap > max_gap_seconds:
                continue
            link = EpisodeLink(
                source_episode_id=prev.episode_id,
                target_episode_id=curr.episode_id,
                link_type=LINK_CONTINUATION,
                source=LINK_SOURCE_INFERRED,
                evidence=f"Same state '{prev.state}', {gap:.0f}s apart",
            )
            self.add_link(link)
            new_links.append(link)

        for i, candidate in enumerate(by_time):
            if candidate.summary_embedding is None:
                continue
            for j in range(i):
                failed = by_time[j]
                if failed.outcome != OUTCOME_FAILURE:
                    continue
                if not candidate.state or failed.state != candidate.state:
                    continue
                if failed.summary_embedding is None:
                    continue
                sim = cosine_similarity(
                    candidate.summary_embedding, failed.summary_embedding
                )
                if sim >= sim_threshold:
                    link = EpisodeLink(
                        source_episode_id=candidate.episode_id,
                        target_episode_id=failed.episode_id,
                        link_type=LINK_RETRY_OF,
                        strength=sim,
                        source=LINK_SOURCE_INFERRED,
                        evidence=f"Retrying failed episode (sim={sim:.2f})",
                    )
                    self.add_link(link)
                    new_links.append(link)

        return new_links

    # -- queries ---------------------------------------------------------------

    def get_links_from(self, episode_id: str) -> list[EpisodeLink]:
        """All outgoing links from an episode."""
        return [self._links[i] for i in self._out_index.get(episode_id, [])]

    def traverse(
        self,
        start_id: str,
        depth: int = 1,
        link_types: Sequence[str] | None = None,
    ) -> list[Episode]:
        """BFS from *start_id* up to *depth* hops.  Returns visited episodes
        (excluding the start node).
        """
        visited: set[str] = set()
        frontier = {start_id}

        for _ in range(depth):
            next_frontier: set[str] = set()
            for nid in frontier:
                for link in self.get_links_from(nid):
                    target = link.target_episode_id
                    if target in visited:
                        continue
                    if link_types and link.link_type not in link_types:
                        continue
                    next_frontier.add(target)
            visited |= frontier
            frontier = next_frontier - visited

        visited |= frontier
        visited.discard(start_id)

        return [self._episodes[eid] for eid in visited if eid in self._episodes]

    def neighbors(
        self,
        episode_id: str,
        link_types: Sequence[str] | None = None,
        depth: int = 1,
    ) -> list[Episode]:
        """Alias kept for backward compatibility with retriever."""
        return self.traverse(episode_id, depth=depth, link_types=link_types)

    def find_similar_by_state(
        self,
        state: str,
        embedding: list[float],
        top_k: int = 5,
    ) -> list[tuple[float, Episode]]:
        """Find episodes sharing *state*, ranked by embedding similarity."""
        scored: list[tuple[float, Episode]] = []
        for ep in self._episodes.values():
            if ep.state != state:
                continue
            if ep.summary_embedding is None:
                continue
            sim = cosine_similarity(embedding, ep.summary_embedding)
            scored.append((sim, ep))
        scored.sort(key=lambda x: x[0], reverse=True)
        return scored[:top_k]

    def get_episode(self, episode_id: str) -> Episode | None:
        return self._episodes.get(episode_id)

    @property
    def episode_ids(self) -> list[str]:
        return list(self._episodes)

    @property
    def links(self) -> list[EpisodeLink]:
        return list(self._links)

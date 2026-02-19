"""Tests for the episodic memory system.

Uses the deterministic hash encoder (no ML model required) so tests are
fast and reproducible.
"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

import pytest

from memory.encoder import Encoder
from memory.episodes import EpisodeBoundaryDetector
from memory.facts import FactExtractor
from memory.graph import EpisodeGraph
from memory.models import (
    Episode,
    LINK_CONTINUATION,
    LINK_LEARNED_FROM,
    LINK_RETRY_OF,
    OUTCOME_FAILURE,
    OUTCOME_SUCCESS,
    OUTCOME_UNKNOWN,
    ROLE_AGENT,
    ROLE_USER,
    RetrievalResult,
    TimelineEntry,
    cosine_similarity,
)
from memory.reinforcer import Reinforcer
from memory.retriever import Retriever
from memory.timeline import Timeline


@pytest.fixture
def encoder() -> Encoder:
    return Encoder()


@pytest.fixture
def timeline() -> Timeline:
    tl = Timeline()
    tl.append(ROLE_USER, "Hello, I need help with my order.")
    tl.append(ROLE_AGENT, "Sure! What's your order number?")
    tl.append(ROLE_USER, "It's 12345.")
    tl.append(ROLE_AGENT, "I found order 12345. It ships tomorrow.")
    tl.append(ROLE_USER, "Can I change the shipping address?")
    tl.append(ROLE_AGENT, "Yes, what's the new address?")
    tl.append(ROLE_USER, "123 Main St, Springfield.")
    tl.append(ROLE_AGENT, "Address updated!")
    return tl


@pytest.fixture
def state_change_timeline() -> Timeline:
    """Timeline with explicit state_label changes to trigger boundaries."""
    tl = Timeline()
    tl.append(ROLE_USER, "Let's plan the architecture.", state_label="planning")
    tl.append(ROLE_AGENT, "I suggest a microservices approach.", state_label="planning")
    tl.append(ROLE_USER, "Now let's debug the auth service.", state_label="debugging")
    tl.append(ROLE_AGENT, "I see a null pointer in line 42.", state_label="debugging")
    tl.append(ROLE_USER, "Fixed it. Let's review the PR.", state_label="review")
    tl.append(ROLE_AGENT, "LGTM, merging now.", state_label="review")
    return tl


# -- Timeline ------------------------------------------------------------------

class TestTimeline:
    def test_append_and_length(self):
        tl = Timeline()
        tl.append(ROLE_USER, "hello")
        tl.append(ROLE_AGENT, "hi")
        assert len(tl) == 2

    def test_last(self, timeline: Timeline):
        last = timeline.last(2)
        assert len(last) == 2
        assert last[-1].raw_content == "Address updated!"

    def test_by_ids(self, timeline: Timeline):
        entries = timeline.entries
        ids = [entries[0].entry_id, entries[2].entry_id]
        result = timeline.by_ids(ids)
        assert len(result) == 2

    def test_sequence_ids_monotonic(self, timeline: Timeline):
        seq_ids = [e.sequence_id for e in timeline.entries]
        assert seq_ids == list(range(len(seq_ids)))

    def test_get_range(self, timeline: Timeline):
        result = timeline.get_range(1, 3)
        assert len(result) == 3
        assert result[0].sequence_id == 1
        assert result[-1].sequence_id == 3

    def test_get_recent(self, timeline: Timeline):
        recent = timeline.get_recent(3)
        assert len(recent) == 3
        assert recent[-1].raw_content == "Address updated!"
        assert recent[0].sequence_id < recent[-1].sequence_id

    def test_append_entry(self):
        tl = Timeline()
        entry = TimelineEntry(role=ROLE_USER, raw_content="pre-built entry")
        tl.append_entry(entry)
        assert len(tl) == 1
        assert tl.entries[0].raw_content == "pre-built entry"
        assert tl.entries[0].sequence_id == 0

    def test_sqlite_persistence(self, tmp_path):
        db = str(tmp_path / "test.db")
        tl1 = Timeline(db_path=db)
        tl1.append(ROLE_USER, "persist me")

        tl2 = Timeline(db_path=db)
        assert len(tl2) == 1
        assert tl2.entries[0].raw_content == "persist me"

    def test_embedding_roundtrip(self):
        tl = Timeline()
        entry = TimelineEntry(
            role=ROLE_USER,
            raw_content="with embedding",
            semantic_embedding=[0.1, 0.2, 0.3],
        )
        tl.append_entry(entry)
        loaded = tl.entries[0]
        assert loaded.semantic_embedding == [0.1, 0.2, 0.3]


# -- Encoder -------------------------------------------------------------------

class TestEncoder:
    def test_deterministic(self, encoder: Encoder):
        a = encoder.encode_text("hello world")
        b = encoder.encode_text("hello world")
        assert a == b

    def test_different_texts(self, encoder: Encoder):
        a = encoder.encode_text("the cat sat on the mat")
        b = encoder.encode_text("quantum mechanics lecture notes")
        sim = cosine_similarity(a, b)
        assert sim < 0.9

    def test_embedding_dimension(self, encoder: Encoder):
        vec = encoder.encode_text("test")
        assert len(vec) == encoder.dim

    def test_state_embedding_deterministic(self, encoder: Encoder):
        a = encoder.encode_state("debugging")
        b = encoder.encode_state("debugging")
        assert a == b

    def test_state_embedding_different_labels(self, encoder: Encoder):
        a = encoder.encode_state("debugging")
        b = encoder.encode_state("planning")
        sim = cosine_similarity(a, b)
        assert sim < 0.9

    def test_state_embedding_cached(self, encoder: Encoder):
        encoder.encode_state("debugging")
        assert "debugging" in encoder._state_cache

    def test_time_decay_now_is_one(self, encoder: Encoder):
        now = datetime.now(timezone.utc)
        assert encoder.time_decay(now, reference=now) == pytest.approx(1.0)

    def test_time_decay_decreases(self, encoder: Encoder):
        ref = datetime.now(timezone.utc)
        old = ref - timedelta(hours=48)
        recent = ref - timedelta(hours=1)
        assert encoder.time_decay(old, ref) < encoder.time_decay(recent, ref)

    def test_time_decay_half_life(self):
        enc = Encoder(time_half_life_hours=24.0)
        ref = datetime.now(timezone.utc)
        one_day_ago = ref - timedelta(hours=24)
        assert enc.time_decay(one_day_ago, ref) == pytest.approx(0.5, abs=0.01)


# -- Episode Boundary Detection ------------------------------------------------

class TestEpisodeBoundaryDetector:
    def test_builds_at_least_one_episode(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        assert len(episodes) >= 1

    def test_episodes_cover_all_entries(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        all_ids = set()
        for ep in episodes:
            all_ids.update(ep.entry_ids)
        assert all_ids == {e.entry_id for e in timeline.entries}

    def test_empty_timeline(self, encoder: Encoder):
        detector = EpisodeBoundaryDetector(encoder)
        assert detector.build_episodes([]) == []

    def test_episode_has_summary(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        for ep in episodes:
            assert ep.summary_text

    def test_state_change_triggers_boundary(
        self, encoder: Encoder, state_change_timeline: Timeline
    ):
        detector = EpisodeBoundaryDetector(encoder)
        entries = state_change_timeline.entries
        boundaries = detector.detect_boundaries(entries)
        assert len(boundaries) >= 2, (
            f"Expected >=2 boundaries for 3 state phases, got {boundaries}"
        )

    def test_state_change_produces_multiple_episodes(
        self, encoder: Encoder, state_change_timeline: Timeline
    ):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(state_change_timeline.entries)
        assert len(episodes) >= 3, (
            f"Expected >=3 episodes for 3 state phases, got {len(episodes)}"
        )

    def test_episode_references_sequence_ids(
        self, encoder: Encoder, state_change_timeline: Timeline
    ):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(state_change_timeline.entries)
        for ep in episodes:
            assert ep.timeline_start <= ep.timeline_end
            assert ep.entry_count == len(ep.entry_ids)

    def test_episode_outcome_starts_unknown(
        self, encoder: Encoder, timeline: Timeline
    ):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        for ep in episodes:
            assert ep.outcome == OUTCOME_UNKNOWN

    def test_episode_has_embedding(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        for ep in episodes:
            assert ep.summary_embedding is not None
            assert len(ep.summary_embedding) == encoder.dim

    def test_no_boundary_when_same_state_similar_content(self, encoder: Encoder):
        tl = Timeline()
        tl.append(ROLE_USER, "Tell me about cats.", state_label="chatting")
        tl.append(ROLE_AGENT, "Cats are great pets.", state_label="chatting")
        tl.append(ROLE_USER, "What breeds are popular?", state_label="chatting")

        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(tl.entries)
        assert len(episodes) == 1

    def test_boundary_score_weights(self, encoder: Encoder):
        detector = EpisodeBoundaryDetector(
            encoder,
            state_change_weight=0.6,
            semantic_shift_weight=0.4,
            threshold=0.5,
        )
        prev = TimelineEntry(role=ROLE_USER, raw_content="x", state_label="a")
        curr = TimelineEntry(role=ROLE_USER, raw_content="x", state_label="b")
        prev.semantic_embedding = encoder.encode_text("x")
        curr.semantic_embedding = encoder.encode_text("x")
        score = detector._boundary_score(prev, curr)
        assert score == pytest.approx(0.6, abs=0.01)

    def test_fallback_summary_without_claude(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder, anthropic_client=None)
        detector._anthropic = None
        episodes = detector.build_episodes(timeline.entries)
        for ep in episodes:
            assert "entries:" in ep.summary_text


# -- Graph ---------------------------------------------------------------------

class TestGraph:
    def test_add_and_retrieve(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        graph = EpisodeGraph()
        for ep in episodes:
            graph.add_episode(ep)
        assert len(graph.episode_ids) == len(episodes)

    def test_continuation_links_same_state(self, encoder: Encoder):
        """Consecutive episodes with the same state get CONTINUATION links."""
        now = datetime.now(timezone.utc)
        ep_a = Episode(
            episode_id="a", state="debugging",
            start_time=now, end_time=now + timedelta(minutes=5),
            summary_embedding=encoder.encode_text("fix bug A"),
        )
        ep_b = Episode(
            episode_id="b", state="debugging",
            start_time=now + timedelta(minutes=6),
            end_time=now + timedelta(minutes=10),
            summary_embedding=encoder.encode_text("fix bug B"),
        )
        graph = EpisodeGraph()
        links = graph.auto_link([ep_a, ep_b])
        cont = [l for l in links if l.link_type == LINK_CONTINUATION]
        assert len(cont) == 1
        assert cont[0].source_episode_id == "a"
        assert cont[0].target_episode_id == "b"

    def test_no_continuation_across_states(self, encoder: Encoder):
        now = datetime.now(timezone.utc)
        ep_a = Episode(
            episode_id="a", state="planning",
            start_time=now, end_time=now + timedelta(minutes=5),
            summary_embedding=encoder.encode_text("plan"),
        )
        ep_b = Episode(
            episode_id="b", state="debugging",
            start_time=now + timedelta(minutes=6),
            end_time=now + timedelta(minutes=10),
            summary_embedding=encoder.encode_text("debug"),
        )
        graph = EpisodeGraph()
        links = graph.auto_link([ep_a, ep_b])
        assert len(links) == 0

    def test_retry_of_link(self, encoder: Encoder):
        """RETRY_OF links a new episode to a prior failed one with same state
        and similar embedding."""
        emb = encoder.encode_text("deploy service")
        now = datetime.now(timezone.utc)
        failed = Episode(
            episode_id="fail", state="deploying",
            start_time=now, end_time=now + timedelta(minutes=5),
            outcome=OUTCOME_FAILURE,
            summary_embedding=emb,
        )
        retry = Episode(
            episode_id="retry", state="deploying",
            start_time=now + timedelta(hours=2),
            end_time=now + timedelta(hours=2, minutes=5),
            summary_embedding=emb,
        )
        graph = EpisodeGraph()
        links = graph.auto_link([failed, retry])
        retry_links = [l for l in links if l.link_type == LINK_RETRY_OF]
        assert len(retry_links) == 1
        assert retry_links[0].source_episode_id == "retry"
        assert retry_links[0].target_episode_id == "fail"

    def test_get_links_from(self, encoder: Encoder):
        now = datetime.now(timezone.utc)
        ep_a = Episode(
            episode_id="a", state="debugging",
            start_time=now, end_time=now + timedelta(minutes=5),
            summary_embedding=encoder.encode_text("x"),
        )
        ep_b = Episode(
            episode_id="b", state="debugging",
            start_time=now + timedelta(minutes=6),
            end_time=now + timedelta(minutes=10),
            summary_embedding=encoder.encode_text("y"),
        )
        graph = EpisodeGraph()
        graph.auto_link([ep_a, ep_b])
        from_a = graph.get_links_from("a")
        assert len(from_a) >= 1
        assert all(l.source_episode_id == "a" for l in from_a)
        assert graph.get_links_from("nonexistent") == []

    def test_traverse_depth(self, encoder: Encoder):
        now = datetime.now(timezone.utc)
        eps = []
        for i in range(4):
            eps.append(Episode(
                episode_id=str(i), state="debugging",
                start_time=now + timedelta(minutes=i * 10),
                end_time=now + timedelta(minutes=i * 10 + 5),
                summary_embedding=encoder.encode_text(f"step {i}"),
            ))
        graph = EpisodeGraph()
        graph.auto_link(eps)

        depth_1 = graph.traverse("0", depth=1)
        depth_2 = graph.traverse("0", depth=2)
        assert len(depth_1) >= 1
        assert len(depth_2) >= len(depth_1)

    def test_traverse_with_link_type_filter(self, encoder: Encoder):
        emb = encoder.encode_text("deploy service")
        now = datetime.now(timezone.utc)
        failed = Episode(
            episode_id="fail", state="deploying",
            start_time=now, end_time=now + timedelta(minutes=5),
            outcome=OUTCOME_FAILURE, summary_embedding=emb,
        )
        cont = Episode(
            episode_id="cont", state="deploying",
            start_time=now + timedelta(minutes=6),
            end_time=now + timedelta(minutes=10),
            summary_embedding=emb,
        )
        graph = EpisodeGraph()
        graph.auto_link([failed, cont])
        only_retry = graph.traverse(
            "cont", depth=1, link_types=[LINK_RETRY_OF]
        )
        only_cont = graph.traverse(
            "fail", depth=1, link_types=[LINK_CONTINUATION]
        )
        assert all(
            graph.get_episode(ep.episode_id) is not None
            for ep in only_retry + only_cont
        )

    def test_find_similar_by_state(self, encoder: Encoder):
        emb_target = encoder.encode_text("fix authentication bug")
        now = datetime.now(timezone.utc)
        ep_match = Episode(
            episode_id="match", state="debugging",
            start_time=now, end_time=now + timedelta(minutes=5),
            summary_embedding=encoder.encode_text("debug auth issue"),
        )
        ep_diff_state = Episode(
            episode_id="wrong_state", state="planning",
            start_time=now, end_time=now + timedelta(minutes=5),
            summary_embedding=encoder.encode_text("debug auth issue"),
        )
        ep_no_emb = Episode(
            episode_id="no_emb", state="debugging",
            start_time=now, end_time=now + timedelta(minutes=5),
        )
        graph = EpisodeGraph()
        for ep in [ep_match, ep_diff_state, ep_no_emb]:
            graph.add_episode(ep)

        results = graph.find_similar_by_state("debugging", emb_target, top_k=5)
        result_ids = [ep.episode_id for _, ep in results]
        assert "match" in result_ids
        assert "wrong_state" not in result_ids
        assert "no_emb" not in result_ids


# -- Facts ---------------------------------------------------------------------

class TestFacts:
    def test_extract_creates_fact(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        extractor = FactExtractor(encoder)
        for ep in episodes:
            extractor.extract(ep)
        assert len(extractor.facts) >= 1

    def test_query_returns_results(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        extractor = FactExtractor(encoder)
        for ep in episodes:
            extractor.extract(ep)
        results = extractor.query("order shipping")
        assert len(results) >= 1

    def test_extract_patterns_batch(self, encoder: Encoder):
        episodes = [
            Episode(episode_id=f"e{i}", summary_text=f"Summary of episode {i}")
            for i in range(4)
        ]
        extractor = FactExtractor(encoder, min_episodes_for_extraction=3)
        extractor.extract_patterns(episodes)
        assert len(extractor.facts) >= 1

    def test_auto_trigger_at_threshold(self, encoder: Encoder):
        extractor = FactExtractor(encoder, min_episodes_for_extraction=3)
        for i in range(3):
            extractor.extract(
                Episode(episode_id=f"e{i}", summary_text=f"Event {i} happened")
            )
        assert len(extractor.facts) >= 1

    def test_below_threshold_no_crash(self, encoder: Encoder):
        extractor = FactExtractor(encoder, min_episodes_for_extraction=5)
        extractor.extract(Episode(episode_id="e0", summary_text="Only one"))
        assert len(extractor.facts) >= 1

    def test_merge_similar_facts(self, encoder: Encoder):
        extractor = FactExtractor(encoder, merge_threshold=0.80)
        ep1 = Episode(episode_id="a", summary_text="exact same text")
        ep2 = Episode(episode_id="b", summary_text="exact same text")
        extractor.extract(ep1)
        extractor.extract(ep2)
        assert len(extractor.facts) == 1
        assert extractor.facts[0].support_count >= 2

    def test_facts_have_embedding(self, encoder: Encoder):
        extractor = FactExtractor(encoder)
        extractor.extract(Episode(episode_id="x", summary_text="test fact"))
        for f in extractor.facts:
            assert f.fact_embedding is not None
            assert len(f.fact_embedding) == encoder.dim


# -- Retriever -----------------------------------------------------------------

class TestRetriever:
    def _build_retriever(self, encoder, episodes):
        """Helper: graph + fact extractor + retriever from a list of episodes."""
        graph = EpisodeGraph()
        graph.auto_link(episodes)
        extractor = FactExtractor(encoder)
        for ep in episodes:
            extractor.extract(ep)
        retriever = Retriever(
            encoder=encoder, graph=graph, fact_extractor=extractor
        )
        return retriever, graph, extractor

    def test_basic_retrieval(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        retriever, _, _ = self._build_retriever(encoder, episodes)

        result = retriever.retrieve("What was the order number?")
        assert len(result.episodes) >= 1
        assert len(result.scores) == len(result.episodes)

    def test_state_filter_prefers_matching_state(self, encoder: Encoder):
        now = datetime.now(timezone.utc)
        emb = encoder.encode_text("auth issue")
        ep_debug = Episode(
            episode_id="dbg", state="debugging",
            start_time=now, end_time=now + timedelta(minutes=5),
            summary_text="fix null pointer",
            summary_embedding=emb,
        )
        ep_plan = Episode(
            episode_id="pln", state="planning",
            start_time=now + timedelta(minutes=10),
            end_time=now + timedelta(minutes=15),
            summary_text="plan auth module",
            summary_embedding=emb,
        )
        retriever, _, _ = self._build_retriever(encoder, [ep_debug, ep_plan])
        result = retriever.retrieve("auth issue", current_state="debugging")
        assert result.episodes[0].episode_id == "dbg"

    def test_failure_boost(self, encoder: Encoder):
        now = datetime.now(timezone.utc)
        emb = encoder.encode_text("deploy the service")
        ep_ok = Episode(
            episode_id="ok", state="deploying",
            start_time=now, end_time=now + timedelta(minutes=5),
            summary_text="deploy service",
            summary_embedding=emb,
            outcome="success",
        )
        ep_fail = Episode(
            episode_id="fail", state="deploying",
            start_time=now + timedelta(minutes=10),
            end_time=now + timedelta(minutes=15),
            summary_text="deploy service",
            summary_embedding=emb,
            outcome=OUTCOME_FAILURE,
        )
        retriever, _, _ = self._build_retriever(encoder, [ep_ok, ep_fail])
        result = retriever.retrieve("deploy the service")
        assert result.episodes[0].episode_id == "fail"

    def test_graph_traversal_expands_results(self, encoder: Encoder):
        now = datetime.now(timezone.utc)
        ep_a = Episode(
            episode_id="a", state="debugging",
            start_time=now, end_time=now + timedelta(minutes=5),
            summary_text="fix bug A",
            summary_embedding=encoder.encode_text("fix bug A"),
        )
        ep_b = Episode(
            episode_id="b", state="debugging",
            start_time=now + timedelta(minutes=6),
            end_time=now + timedelta(minutes=10),
            summary_text="fix bug B (unrelated text)",
            summary_embedding=encoder.encode_text("completely different topic xyz"),
        )
        retriever_trav, _, _ = self._build_retriever(encoder, [ep_a, ep_b])
        result_with = retriever_trav.retrieve("fix bug A", traverse=True)

        graph_no = EpisodeGraph()
        for ep in [ep_a, ep_b]:
            graph_no.add_episode(ep)
        ext_no = FactExtractor(encoder)
        retriever_no = Retriever(
            encoder=encoder, graph=graph_no, fact_extractor=ext_no,
            top_k_episodes=1,
        )
        result_without = retriever_no.retrieve("fix bug A", traverse=False)

        with_ids = {ep.episode_id for ep in result_with.episodes}
        without_ids = {ep.episode_id for ep in result_without.episodes}
        assert with_ids >= without_ids

    def test_narrative_is_populated(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        retriever, _, _ = self._build_retriever(encoder, episodes)

        result = retriever.retrieve("order number")
        assert result.causal_narrative
        assert "→" in result.causal_narrative or len(result.episodes) == 1

    def test_retrieve_without_state(self, encoder: Encoder, timeline: Timeline):
        """Backward-compat: retrieve(query) still works with no state."""
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        retriever, _, _ = self._build_retriever(encoder, episodes)

        result = retriever.retrieve("order")
        assert len(result.episodes) >= 1


# -- Reinforcer ----------------------------------------------------------------

class TestReinforcer:
    def _build_stack(self, encoder, episodes):
        graph = EpisodeGraph()
        graph.auto_link(episodes)
        extractor = FactExtractor(encoder)
        for ep in episodes:
            extractor.extract(ep)
        retriever = Retriever(
            encoder=encoder, graph=graph, fact_extractor=extractor
        )
        reinforcer = Reinforcer(
            encoder=encoder, graph=graph, fact_extractor=extractor
        )
        return retriever, reinforcer, graph, extractor

    def test_reinforce_creates_outcome_episode(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        retriever, reinforcer, graph, _ = self._build_stack(encoder, episodes)

        result = retriever.retrieve("shipping address")
        before_count = len(graph.episode_ids)
        outcome_ep = reinforcer.reinforce(result, OUTCOME_SUCCESS, "All good")
        assert outcome_ep is not None
        assert len(graph.episode_ids) == before_count + 1
        assert outcome_ep.outcome == OUTCOME_SUCCESS

    def test_reinforce_links_to_retrieved(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        retriever, reinforcer, graph, _ = self._build_stack(encoder, episodes)

        result = retriever.retrieve("shipping address")
        outcome_ep = reinforcer.reinforce(result, OUTCOME_SUCCESS)
        links = graph.get_links_from(outcome_ep.episode_id)
        assert len(links) == len(result.episodes)
        assert all(l.link_type == LINK_LEARNED_FROM for l in links)

    def test_reinforce_updates_episode_scores(self, encoder: Encoder, timeline: Timeline):
        detector = EpisodeBoundaryDetector(encoder)
        episodes = detector.build_episodes(timeline.entries)
        retriever, reinforcer, graph, _ = self._build_stack(encoder, episodes)

        result = retriever.retrieve("shipping address")
        reinforcer.reinforce(result, OUTCOME_SUCCESS)
        for ep in result.episodes:
            stored = graph.get_episode(ep.episode_id)
            if stored:
                assert stored.outcome_score != 0.0

    def test_reinforce_failure_contradicts_facts(self, encoder: Encoder):
        ep = Episode(
            episode_id="ep1", summary_text="Always restart before deploying",
            summary_embedding=encoder.encode_text("Always restart before deploying"),
            state="deploying",
        )
        _, reinforcer, graph, extractor = self._build_stack(encoder, [ep])

        facts_before = extractor.facts
        initial_contradiction = (
            facts_before[0].contradiction_count if facts_before else 0
        )

        result = RetrievalResult(
            episodes=[ep],
            facts=list(extractor.facts),
            scores=[1.0],
        )
        reinforcer.reinforce(result, OUTCOME_FAILURE, "Deploy crashed")

        for f in extractor.facts:
            assert f.contradiction_count > initial_contradiction

    def test_reinforce_success_supports_facts(self, encoder: Encoder):
        ep = Episode(
            episode_id="ep1", summary_text="Check logs before restarting",
            summary_embedding=encoder.encode_text("Check logs before restarting"),
            state="debugging",
        )
        _, reinforcer, graph, extractor = self._build_stack(encoder, [ep])

        facts_before = extractor.facts
        initial_support = facts_before[0].support_count if facts_before else 0

        result = RetrievalResult(
            episodes=[ep],
            facts=list(extractor.facts),
            scores=[1.0],
        )
        reinforcer.reinforce(result, OUTCOME_SUCCESS, "Issue resolved")

        for f in extractor.facts:
            assert f.support_count > initial_support

    def test_reinforce_empty_result(self, encoder: Encoder):
        graph = EpisodeGraph()
        extractor = FactExtractor(encoder)
        reinforcer = Reinforcer(
            encoder=encoder, graph=graph, fact_extractor=extractor
        )
        result = RetrievalResult()
        assert reinforcer.reinforce(result, OUTCOME_SUCCESS) is None

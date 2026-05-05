"""Head-to-head comparison: episodic memory vs flat vector RAG.

Runs BOTH agents through ALL 9 rounds of the debugging scenario and prints
a rich table for every round showing:

  - What the episodic agent retrieved (episodes with outcomes, facts)
  - What the baseline agent retrieved (flat text chunks)
  - Each agent's full reasoning and short decision
  - Whether that decision was correct

When ANTHROPIC_API_KEY is set, both agents call Claude (claude-sonnet-4-20250514)
with different context formats.  Falls back to keyword heuristics otherwise.

Usage:
    python -m eval.compare
"""

from __future__ import annotations

import logging
import os
import textwrap
from dataclasses import dataclass, field
from typing import Any

from agent.agent import EpisodicAgent
from baseline.vector_agent import VectorAgent
from memory.encoder import Encoder
from memory.episodes import EpisodeBoundaryDetector
from memory.facts import FactExtractor
from memory.graph import EpisodeGraph
from memory.models import (
    OUTCOME_FAILURE,
    OUTCOME_SUCCESS,
    OUTCOME_UNKNOWN,
    ROLE_AGENT,
    ROLE_SYSTEM,
    ROLE_USER,
    RetrievalResult,
    cosine_similarity,
)
from memory.reinforcer import Reinforcer
from memory.retriever import Retriever
from memory.timeline import Timeline
from simulator.scenarios import DebugRound, debugging_scenario

log = logging.getLogger(__name__)

W = 80
BOX_H = "=" * W
COL_W = 38

_POOL_PATTERNS = [
    "connection pool", "pool exhaust", "pool saturat",
    "pool starvation", "pool contention",
]


# -- stacks ----------------------------------------------------------------

class EpisodicStack:
    """Memory management wrapper -- delegates decisions to EpisodicAgent."""

    def __init__(self, encoder: Encoder, anthropic_client: Any = None) -> None:
        self.encoder = encoder
        self.timeline = Timeline()
        self.detector = EpisodeBoundaryDetector(encoder)
        self.graph = EpisodeGraph()
        self.facts = FactExtractor(encoder, min_episodes_for_extraction=3)
        self.retriever = Retriever(
            encoder=encoder, graph=self.graph, fact_extractor=self.facts,
        )
        self.reinforcer = Reinforcer(
            encoder=encoder, graph=self.graph, fact_extractor=self.facts,
        )
        self.agent = EpisodicAgent(
            encoder=encoder, anthropic_client=anthropic_client,
        )

    def ingest_round(self, rd: DebugRound) -> None:
        self.timeline.append(ROLE_USER, rd.bug_report, state_label=rd.state_label)
        for note in rd.investigation_notes:
            self.timeline.append(ROLE_AGENT, note, state_label=rd.state_label)
        if rd.attempted_fix:
            self.timeline.append(
                ROLE_AGENT, f"Fix attempted: {rd.attempted_fix}",
                state_label=rd.state_label,
            )
        self.timeline.append(
            ROLE_SYSTEM,
            f"Outcome: {rd.outcome}. Root cause: {rd.actual_root_cause}",
            state_label=rd.state_label,
        )
        if rd.correction:
            self.timeline.append(
                ROLE_SYSTEM,
                f"Correction: {rd.correction}",
                state_label=rd.state_label,
            )
        episodes = self.detector.build_episodes(self.timeline.entries)
        for ep in episodes:
            if ep.episode_id not in self.graph.episode_ids:
                self.graph.add_episode(ep)
                self.facts.extract(ep)
        self.graph.auto_link(episodes)
        if episodes:
            latest = episodes[-1]
            latest.outcome = rd.outcome
            latest.outcome_details = rd.actual_root_cause
            if rd.assumption:
                latest.assumptions = [rd.assumption]
            if rd.correction:
                latest.corrections = [rd.correction]
            result = RetrievalResult(episodes=[latest], facts=[], scores=[1.0])
            outcome_ep = self.reinforcer.reinforce(
                result, rd.outcome, outcome_details=rd.actual_root_cause,
            )
            if outcome_ep is not None:
                if rd.assumption:
                    outcome_ep.assumptions = [rd.assumption]
                if rd.correction:
                    outcome_ep.corrections = [rd.correction]

    def retrieve(self, query: str) -> RetrievalResult:
        return self.retriever.retrieve(query, current_state="debugging")

    def trigger_fact_extraction(self) -> None:
        real = [
            self.graph.get_episode(eid)
            for eid in self.graph.episode_ids
        ]
        self.facts.extract_patterns(
            [ep for ep in real if ep is not None and ep.summary_text]
        )


class BaselineStack:
    """Memory management wrapper -- delegates decisions to VectorAgent."""

    def __init__(self, encoder: Encoder, anthropic_client: Any = None) -> None:
        self.encoder = encoder
        self._store: list[tuple[list[float], str]] = []
        self.agent = VectorAgent(
            encoder=encoder, anthropic_client=anthropic_client,
        )

    def ingest_round(self, rd: DebugRound) -> None:
        texts = (
            [rd.bug_report]
            + rd.investigation_notes
            + ([f"Fix attempted: {rd.attempted_fix}"] if rd.attempted_fix else [])
            + [f"Outcome: {rd.outcome}. Root cause: {rd.actual_root_cause}"]
            + ([f"Correction: {rd.correction}"] if rd.correction else [])
        )
        for t in texts:
            self._store.append((self.encoder.encode_text(t), t))

    def retrieve(self, query: str, top_k: int = 5) -> list[str]:
        if not self._store:
            return []
        q_emb = self.encoder.encode_text(query)
        scored = [
            (cosine_similarity(q_emb, emb), text)
            for emb, text in self._store
        ]
        scored.sort(key=lambda x: x[0], reverse=True)
        return [text for _, text in scored[:top_k]]


# -- per-round result ------------------------------------------------------

@dataclass
class RoundResult:
    round_num: int
    service: str
    bug_report: str
    ground_truth: str
    round_label: str = ""

    ep_episodes: list[str] = field(default_factory=list)
    ep_facts: list[str] = field(default_factory=list)
    ep_decision: str = ""
    ep_reasoning: str = ""
    ep_correct: bool = False

    bl_chunks: list[str] = field(default_factory=list)
    bl_decision: str = ""
    bl_reasoning: str = ""
    bl_correct: bool = False

    ep_keyword_hits: int = 0
    bl_keyword_hits: int = 0
    total_keywords: int = 0

    ep_outcome_labeled: int = 0
    ep_total_retrieved: int = 0
    bl_outcome_chunks: int = 0
    bl_total_retrieved: int = 0


# -- correctness -----------------------------------------------------------

def _is_correct(decision: str, rd: DebugRound) -> bool:
    low = decision.lower()
    if rd.incorrect_keywords:
        if any(kw.lower() in low for kw in rd.incorrect_keywords):
            return False
    if rd.correct_keywords:
        return any(kw.lower() in low for kw in rd.correct_keywords)
    return any(p in low for p in _POOL_PATTERNS)


# -- formatting helpers ----------------------------------------------------

def _wrap_block(text: str, width: int = COL_W) -> list[str]:
    lines: list[str] = []
    for paragraph in text.splitlines():
        if not paragraph.strip():
            lines.append("")
        else:
            lines.extend(textwrap.wrap(paragraph, width=width))
    return lines or [""]


def _side_by_side(label: str, left: str, right: str) -> str:
    l_lines = _wrap_block(left)
    r_lines = _wrap_block(right)
    n = max(len(l_lines), len(r_lines))
    l_lines += [""] * (n - len(l_lines))
    r_lines += [""] * (n - len(r_lines))
    out = [f"  {label}"]
    for ll, rl in zip(l_lines, r_lines):
        out.append(f"  {ll:<{COL_W}}  |  {rl:<{COL_W}}")
    return "\n".join(out)


def _count_hits(text: str, keywords: list[str]) -> int:
    low = text.lower()
    return sum(1 for kw in keywords if kw.lower() in low)


# -- main runner -----------------------------------------------------------

def run_comparison() -> list[RoundResult]:
    encoder = Encoder()
    anthropic_client = _try_init_anthropic()
    mode = "Claude" if anthropic_client else "heuristic"

    episodic = EpisodicStack(encoder, anthropic_client=anthropic_client)
    baseline = BaselineStack(encoder, anthropic_client=anthropic_client)
    rounds = debugging_scenario()
    results: list[RoundResult] = []

    _print_header(mode)

    for rd in rounds:
        ep_result = episodic.retrieve(rd.bug_report)
        bl_entries = baseline.retrieve(rd.bug_report)

        ep_dr = episodic.agent.decide(rd.bug_report, ep_result)
        bl_dr = baseline.agent.decide(rd.bug_report, bl_entries)

        ep_correct = _is_correct(ep_dr.decision, rd)
        bl_correct = _is_correct(bl_dr.decision, rd)

        combined_ep = ep_dr.context_used + " " + ep_dr.decision
        combined_bl = bl_dr.context_used + " " + bl_dr.decision
        ep_kw = _count_hits(combined_ep, rd.expected_keywords)
        bl_kw = _count_hits(combined_bl, rd.expected_keywords)

        ep_top = ep_result.episodes[:5]
        ep_outcome_labeled = sum(
            1 for ep in ep_top
            if ep.outcome in (OUTCOME_SUCCESS, OUTCOME_FAILURE)
        )
        bl_top = bl_entries[:5]
        bl_outcome_chunks = sum(
            1 for c in bl_top if "outcome:" in c.lower()
        )

        rr = RoundResult(
            round_num=rd.round_num,
            service=rd.service,
            bug_report=rd.bug_report,
            ground_truth=rd.actual_root_cause,
            round_label=rd.round_label,
            ep_episodes=[
                f"[{ep.outcome.upper()}] {ep.summary_text[:60]}"
                + (f"  => {'; '.join(ep.corrections)}" if ep.corrections else "")
                for ep in ep_top
            ],
            ep_facts=[
                f"{f.fact_text[:70]} [sup={f.support_count}]"
                for f in ep_result.facts[:3]
            ],
            ep_decision=ep_dr.decision,
            ep_reasoning=ep_dr.reasoning,
            ep_correct=ep_correct,
            bl_chunks=[e[:75] for e in bl_top],
            bl_decision=bl_dr.decision,
            bl_reasoning=bl_dr.reasoning,
            bl_correct=bl_correct,
            ep_keyword_hits=ep_kw,
            bl_keyword_hits=bl_kw,
            total_keywords=len(rd.expected_keywords),
            ep_outcome_labeled=ep_outcome_labeled,
            ep_total_retrieved=len(ep_top),
            bl_outcome_chunks=bl_outcome_chunks,
            bl_total_retrieved=len(bl_top),
        )
        results.append(rr)
        _print_round(rr)

        episodic.ingest_round(rd)
        baseline.ingest_round(rd)

        if rd.trigger_fact_extraction:
            print(
                f"\n  >>> TRIGGERING FACT EXTRACTION "
                f"AFTER ROUND {rd.round_num} <<<"
            )
            episodic.trigger_fact_extraction()
            for f in episodic.facts.facts:
                print(f"     * {f.fact_text[:80]}")
            print()

    _print_summary_table(results)
    _print_memory_state(episodic)
    return results


# -- printing --------------------------------------------------------------

def _print_header(mode: str) -> None:
    print(f"\n{BOX_H}")
    print("  EPISODIC MEMORY  vs  FLAT VECTOR RAG")
    print("  Scenario: 9-Round Debugging")
    print(f"  Decision mode: {mode}")
    print(BOX_H)
    print("  Round types:")
    print("    LEARN       -- agent builds memory from failure")
    print("    RED HERRING -- root cause is NOT connection pool")
    print("    TEST        -- root cause IS connection pool")
    print("    SUBTLE      -- connection pool but different symptoms")
    print("    CORRECTION  -- agent gets corrected, then tested on similar case")
    print(BOX_H)


def _print_round(rr: RoundResult) -> None:
    label = f" [{rr.round_label}]" if rr.round_label else ""
    print(f"\n+{'-' * (W - 2)}+")
    print(f"|{' ' * (W - 2)}|")
    title = f"ROUND {rr.round_num}{label}  |  {rr.service}"
    print(f"|  {title:<{W - 4}}|")
    bug = rr.bug_report
    if len(bug) > W - 4:
        bug = bug[: W - 7] + "..."
    print(f"|  {bug:<{W - 4}}|")
    print(f"|{' ' * (W - 2)}|")
    print(f"+{'-' * (COL_W + 2)}+{'-' * (W - COL_W - 5)}+")

    header = (
        f"|  {'EPISODIC AGENT':<{COL_W}}"
        f"|  {'BASELINE AGENT':<{W - COL_W - 5}}|"
    )
    print(header)
    print(f"+{'-' * (COL_W + 2)}+{'-' * (W - COL_W - 5)}+")

    ep_ep_text = (
        "\n".join(rr.ep_episodes) if rr.ep_episodes else "(no prior episodes)"
    )
    bl_ch_text = (
        "\n".join(f"* {c}" for c in rr.bl_chunks) if rr.bl_chunks
        else "(no prior entries)"
    )
    print(_side_by_side("Retrieved memory:", ep_ep_text, bl_ch_text))

    if rr.ep_facts:
        ep_facts_text = "\n".join(rr.ep_facts)
        print(_side_by_side(
            "Generalized facts:",
            ep_facts_text,
            "(N/A -- no fact extraction)",
        ))

    print(f"  {'-' * COL_W}--+--{'-' * (W - COL_W - 6)}")
    print(_side_by_side("Reasoning:", rr.ep_reasoning, rr.bl_reasoning))

    print(f"  {'-' * COL_W}--+--{'-' * (W - COL_W - 6)}")

    ep_mark = "CORRECT" if rr.ep_correct else "WRONG"
    bl_mark = "CORRECT" if rr.bl_correct else "WRONG"
    ep_dec = textwrap.shorten(rr.ep_decision, width=COL_W, placeholder="...")
    bl_dec = textwrap.shorten(
        rr.bl_decision, width=W - COL_W - 6, placeholder="...",
    )
    print(f"  {ep_dec:<{COL_W}}  |  {bl_dec}")
    print(f"  {ep_mark:<{COL_W}}  |  {bl_mark}")

    if rr.total_keywords:
        ep_kw = f"Keywords: {rr.ep_keyword_hits}/{rr.total_keywords}"
        bl_kw = f"Keywords: {rr.bl_keyword_hits}/{rr.total_keywords}"
        print(f"  {ep_kw:<{COL_W}}  |  {bl_kw}")

    print(f"+{'-' * (W - 2)}+")
    gt = rr.ground_truth
    if len(gt) > W - 18:
        gt = gt[: W - 21] + "..."
    print(f"|  Ground truth: {gt:<{W - 18}}|")
    print(f"+{'-' * (W - 2)}+")


def _has_pool_pattern(decision: str) -> bool:
    low = decision.lower()
    return any(p in low for p in _POOL_PATTERNS)


def _print_summary_table(results: list[RoundResult]) -> None:
    # -- 1. Decision correctness table (existing) -------------------------
    print(f"\n{BOX_H}")
    print("  SUMMARY -- Decision Correctness Across All Rounds")
    print(BOX_H)
    print(
        f"  {'Round':<8}{'Type':<14}{'Service':<22}"
        f"{'Episodic':^10}{'Baseline':^10}{'Keywords (E/B)':>16}"
    )
    print(f"  {'-' * 78}")

    ep_wins, bl_wins = 0, 0
    for rr in results:
        label = rr.round_label or "LEARN"
        ep_mark = " OK " if rr.ep_correct else " X  "
        bl_mark = " OK " if rr.bl_correct else " X  "
        kw = (
            f"{rr.ep_keyword_hits}/{rr.total_keywords}"
            f"  {rr.bl_keyword_hits}/{rr.total_keywords}"
            if rr.total_keywords else "    --"
        )
        print(
            f"  {rr.round_num:<8}{label:<14}{rr.service:<22}"
            f"{ep_mark:^10}{bl_mark:^10}{kw:>16}"
        )
        ep_wins += rr.ep_correct
        bl_wins += rr.bl_correct

    print(f"  {'-' * 78}")
    print(f"  {'TOTAL':<44}{ep_wins:^10}{bl_wins:^10}")
    print()

    # -- 2. Structured retrieval score ------------------------------------
    print(BOX_H)
    print("  STRUCTURED RETRIEVAL SCORE")
    print("  (items with explicit outcome labels vs raw outcome mentions)")
    print(BOX_H)
    print(
        f"  {'Round':<8}{'Type':<14}"
        f"{'Episodic (labeled/total)':>26}"
        f"{'Baseline (outcome/total)':>28}"
    )
    print(f"  {'-' * 74}")

    ep_labeled_total, ep_ret_total = 0, 0
    bl_out_total, bl_ret_total = 0, 0
    for rr in results:
        label = rr.round_label or "LEARN"
        if rr.ep_total_retrieved:
            ep_ratio = f"{rr.ep_outcome_labeled}/{rr.ep_total_retrieved}"
        else:
            ep_ratio = "  --"
        if rr.bl_total_retrieved:
            bl_ratio = f"{rr.bl_outcome_chunks}/{rr.bl_total_retrieved}"
        else:
            bl_ratio = "  --"
        print(
            f"  {rr.round_num:<8}{label:<14}"
            f"{ep_ratio:>26}{bl_ratio:>28}"
        )
        ep_labeled_total += rr.ep_outcome_labeled
        ep_ret_total += rr.ep_total_retrieved
        bl_out_total += rr.bl_outcome_chunks
        bl_ret_total += rr.bl_total_retrieved

    print(f"  {'-' * 74}")
    ep_pct = (
        f"{ep_labeled_total}/{ep_ret_total} "
        f"({ep_labeled_total * 100 // ep_ret_total}%)"
        if ep_ret_total else "--"
    )
    bl_pct = (
        f"{bl_out_total}/{bl_ret_total} "
        f"({bl_out_total * 100 // bl_ret_total}%)"
        if bl_ret_total else "--"
    )
    print(f"  {'TOTAL':<22}{ep_pct:>26}{bl_pct:>28}")
    print()

    # -- 3. Pattern application (rounds 4-9) ------------------------------
    print(BOX_H)
    print("  PATTERN APPLICATION (Rounds 4-9)")
    print("  Connection pool pattern: correct when IS root cause, FP when NOT")
    print(BOX_H)
    print(
        f"  {'Round':<8}{'Type':<14}{'Pool correct?':<16}"
        f"{'Episodic':^16}{'Baseline':^16}"
    )
    print(f"  {'-' * 68}")

    ep_fp, bl_fp = 0, 0
    ep_correct_applies, bl_correct_applies = 0, 0
    pattern_rounds = [rr for rr in results if rr.round_num >= 4]
    for rr in pattern_rounds:
        label = rr.round_label or "LEARN"
        pool_is_correct = label not in ("RED HERRING", "CORRECTION")

        ep_applied = _has_pool_pattern(rr.ep_decision)
        bl_applied = _has_pool_pattern(rr.bl_decision)

        if pool_is_correct:
            ep_tag = "correct" if ep_applied else "missed"
            bl_tag = "correct" if bl_applied else "missed"
            ep_correct_applies += ep_applied
            bl_correct_applies += bl_applied
        else:
            ep_tag = "FALSE POS" if ep_applied else "avoided"
            bl_tag = "FALSE POS" if bl_applied else "avoided"
            ep_fp += ep_applied
            bl_fp += bl_applied

        print(
            f"  {rr.round_num:<8}{label:<14}"
            f"{'yes' if pool_is_correct else 'NO':<16}"
            f"{ep_tag:^16}{bl_tag:^16}"
        )

    print(f"  {'-' * 68}")
    print(
        f"  {'Correct applies:':<38}"
        f"{ep_correct_applies:^16}{bl_correct_applies:^16}"
    )
    print(
        f"  {'False positives:':<38}"
        f"{ep_fp:^16}{bl_fp:^16}"
    )
    print()

    # -- 4. Adaptation score (Round 8 → 9) --------------------------------
    rr8 = next((r for r in results if r.round_num == 8), None)
    rr9 = next((r for r in results if r.round_num == 9), None)

    print(BOX_H)
    print("  ADAPTATION SCORE (Round 8 → 9: did the agent learn from correction?)")
    print(BOX_H)

    ep_adapted, bl_adapted = False, False
    if rr8 and rr9:
        ep_pool_r8 = _has_pool_pattern(rr8.ep_decision)
        ep_pool_r9 = _has_pool_pattern(rr9.ep_decision)
        bl_pool_r8 = _has_pool_pattern(rr8.bl_decision)
        bl_pool_r9 = _has_pool_pattern(rr9.bl_decision)

        ep_adapted = ep_pool_r8 and not ep_pool_r9
        bl_adapted = bl_pool_r8 and not bl_pool_r9

        def _adapt_verdict(applied_r8: bool, applied_r9: bool) -> str:
            if applied_r8 and applied_r9:
                return "repeated mistake"
            if applied_r8 and not applied_r9:
                return "ADAPTED"
            return "correctly avoided"

        ep_adapt_tag = _adapt_verdict(ep_pool_r8, ep_pool_r9)
        bl_adapt_tag = _adapt_verdict(bl_pool_r8, bl_pool_r9)

        print(f"  {'':38}{'Episodic':^16}{'Baseline':^16}")
        print(f"  {'-' * 68}")
        print(
            f"  {'Round 8 applied pool pattern:':<38}"
            f"{'yes' if ep_pool_r8 else 'no':^16}"
            f"{'yes' if bl_pool_r8 else 'no':^16}"
        )
        print(
            f"  {'Round 9 applied pool pattern:':<38}"
            f"{'yes' if ep_pool_r9 else 'no':^16}"
            f"{'yes' if bl_pool_r9 else 'no':^16}"
        )
        print(
            f"  {'Verdict:':<38}"
            f"{ep_adapt_tag:^16}{bl_adapt_tag:^16}"
        )
    else:
        print("  (Rounds 8-9 not found — skipping adaptation check)")
    print()

    # -- 5. Weighted total score ------------------------------------------
    ep_avoid_fp = len([
        rr for rr in pattern_rounds
        if rr.round_label in ("RED HERRING", "CORRECTION")
        and not _has_pool_pattern(rr.ep_decision)
    ])
    bl_avoid_fp = len([
        rr for rr in pattern_rounds
        if rr.round_label in ("RED HERRING", "CORRECTION")
        and not _has_pool_pattern(rr.bl_decision)
    ])
    ep_adapt_pts = 2 if ep_adapted else 0
    bl_adapt_pts = 2 if bl_adapted else 0

    ep_total = ep_wins + ep_avoid_fp + ep_adapt_pts
    bl_total = bl_wins + bl_avoid_fp + bl_adapt_pts

    print(BOX_H)
    print("  WEIGHTED SCORE")
    print("  Correct decision = 1pt | Avoided false positive = 1pt | Adaptation = 2pt")
    print(BOX_H)
    print(f"  {'':38}{'Episodic':^16}{'Baseline':^16}")
    print(f"  {'-' * 68}")
    print(f"  {'Correct decisions (×1):':<38}{ep_wins:^16}{bl_wins:^16}")
    print(
        f"  {'Avoided false positives (×1):':<38}"
        f"{ep_avoid_fp:^16}{bl_avoid_fp:^16}"
    )
    print(
        f"  {'Adaptation bonus (×2):':<38}"
        f"{ep_adapt_pts:^16}{bl_adapt_pts:^16}"
    )
    print(f"  {'-' * 68}")
    print(f"  {'WEIGHTED TOTAL:':<38}{ep_total:^16}{bl_total:^16}")
    print()

    if ep_total > bl_total:
        print("  ==>  Episodic memory outperforms flat vector RAG.")
    elif ep_total == bl_total:
        print(
            "  ==>  Tie -- try with ANTHROPIC_API_KEY set "
            "for Claude-based decisions."
        )
    else:
        print("  ==>  Baseline wins (unexpected -- check scenario data).")
    print()


def _print_memory_state(episodic: EpisodicStack) -> None:
    print(BOX_H)
    print("  FINAL EPISODIC MEMORY STATE")
    print(BOX_H)
    print(f"  Episodes: {len(episodic.graph.episode_ids)}")
    print(f"  Links:    {len(episodic.graph.links)}")
    print(f"  Facts:    {len(episodic.facts.facts)}")
    for f in episodic.facts.facts:
        print(
            f"    * {f.fact_text[:72]}  "
            f"[sup={f.support_count} con={f.contradiction_count}]"
        )
    print()


# -- helpers ---------------------------------------------------------------

def _try_init_anthropic() -> Any:
    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        return None
    try:
        import anthropic
        return anthropic.Anthropic(api_key=api_key)
    except Exception:
        return None


# -- entry point -----------------------------------------------------------

if __name__ == "__main__":
    logging.basicConfig(level=logging.WARNING)
    run_comparison()

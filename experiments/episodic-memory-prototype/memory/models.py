"""All data classes for the episodic memory system.

Plain dataclasses, no ORMs.  Designed for SQLite persistence + dict-based
in-memory graph.  Embeddings are list[float]; enum-like values are plain
strings grouped as module-level constants.
"""

from __future__ import annotations

import uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone

import numpy as np


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def _uid() -> str:
    return uuid.uuid4().hex[:12]


def _now() -> datetime:
    return datetime.now(timezone.utc)


def cosine_similarity(a: list[float], b: list[float]) -> float:
    va, vb = np.asarray(a), np.asarray(b)
    denom = np.linalg.norm(va) * np.linalg.norm(vb)
    if denom == 0:
        return 0.0
    return float(np.dot(va, vb) / denom)


# ---------------------------------------------------------------------------
# String enums
# ---------------------------------------------------------------------------

# Roles
ROLE_USER = "user"
ROLE_AGENT = "agent"
ROLE_SYSTEM = "system"

# Content types
CONTENT_TEXT = "text"
CONTENT_TOOL_CALL = "tool_call"
CONTENT_TOOL_RESULT = "tool_result"
CONTENT_ERROR = "error"
CONTENT_FEEDBACK = "feedback"
CONTENT_STATE_CHANGE = "state_change"
CONTENT_DECISION = "decision"
CONTENT_CODE_DIFF = "code_diff"

# Episode types
EPISODE_TASK = "task"
EPISODE_CONVERSATION = "conversation"
EPISODE_INCIDENT = "incident"
EPISODE_PLANNING = "planning"
EPISODE_DEBUGGING = "debugging"
EPISODE_REVIEW = "review"
EPISODE_EXPLORATION = "exploration"

# Outcomes
OUTCOME_SUCCESS = "success"
OUTCOME_FAILURE = "failure"
OUTCOME_PARTIAL = "partial"
OUTCOME_ABANDONED = "abandoned"
OUTCOME_UNKNOWN = "unknown"

# Link types
LINK_CAUSED_BY = "caused_by"
LINK_LED_TO = "led_to"
LINK_REFINED = "refined"
LINK_CONTRADICTED = "contradicted"
LINK_SIMILAR_SITUATION = "similar_situation"
LINK_RETRY_OF = "retry_of"
LINK_ESCALATION_OF = "escalation_of"
LINK_CONTINUATION = "continuation"
LINK_ROLLBACK_OF = "rollback_of"
LINK_LEARNED_FROM = "learned_from"
LINK_TEMPORAL = "temporal"
LINK_SIMILARITY = "similarity"
LINK_CAUSAL = "causal"
LINK_CORRECTION = "correction"

# Link source
LINK_SOURCE_EXPLICIT = "explicit"
LINK_SOURCE_INFERRED = "inferred"
LINK_SOURCE_REINFORCED = "reinforced"

# Fact types
FACT_HEURISTIC = "heuristic"
FACT_CAUSAL_PATTERN = "causal_pattern"
FACT_ANTI_PATTERN = "anti_pattern"
FACT_PREFERENCE = "preference"
FACT_INVARIANT = "invariant"
FACT_CORRELATION = "correlation"


# ---------------------------------------------------------------------------
# Data classes
# ---------------------------------------------------------------------------

@dataclass
class TimelineEntry:
    """Immutable. Append-only. Never modified after creation."""

    entry_id: str = field(default_factory=_uid)
    timestamp: datetime = field(default_factory=_now)
    sequence_id: int = 0

    role: str = ROLE_USER
    content_type: str = CONTENT_TEXT
    raw_content: str = ""

    semantic_embedding: list[float] | None = None
    state_embedding: list[float] | None = None

    agent_id: str = ""
    session_id: str = ""
    state_label: str = ""
    tags: list[str] = field(default_factory=list)


@dataclass
class Episode:
    """A coherent slice of experience.  References timeline entries by ID."""

    episode_id: str = field(default_factory=_uid)

    timeline_start: int = 0
    timeline_end: int = 0
    start_time: datetime = field(default_factory=_now)
    end_time: datetime = field(default_factory=_now)

    state: str = ""
    episode_type: str = EPISODE_TASK

    summary_embedding: list[float] | None = None
    summary_text: str = ""

    outcome: str = OUTCOME_UNKNOWN
    outcome_signal: str = ""
    outcome_details: str = ""
    outcome_score: float = 0.0

    decisions_made: list[str] = field(default_factory=list)
    assumptions: list[str] = field(default_factory=list)
    corrections: list[str] = field(default_factory=list)

    parent_episode_id: str = ""
    triggered_by: str = ""

    entry_count: int = 0
    entry_ids: list[str] = field(default_factory=list)


@dataclass
class EpisodeLink:
    """Typed, weighted, directional relationship between episodes."""

    link_id: str = field(default_factory=_uid)
    source_episode_id: str = ""
    target_episode_id: str = ""

    link_type: str = LINK_TEMPORAL
    strength: float = 1.0
    source: str = LINK_SOURCE_INFERRED
    evidence: str = ""

    created_at: datetime = field(default_factory=_now)
    last_reinforced: datetime = field(default_factory=_now)
    reinforcement_count: int = 0
    decay_eligible: bool = True


@dataclass
class GeneralizedFact:
    """A pattern extracted from multiple episodes."""

    fact_id: str = field(default_factory=_uid)

    fact_text: str = ""
    fact_type: str = FACT_HEURISTIC

    domain: str = ""
    applicable_states: list[str] = field(default_factory=list)

    supporting_episodes: list[str] = field(default_factory=list)
    contradicting_episodes: list[str] = field(default_factory=list)
    support_count: int = 1
    contradiction_count: int = 0

    formed_at: datetime = field(default_factory=_now)
    last_updated: datetime = field(default_factory=_now)
    version: int = 1

    fact_embedding: list[float] | None = None


@dataclass
class MemoryQuery:
    """A memory cue — richer than a search string."""

    query_text: str = ""
    query_embedding: list[float] | None = None

    current_state: str = ""
    current_task: str = ""

    max_episodes: int = 5
    max_facts: int = 3
    recency_weight: float = 0.3
    traverse_links: bool = True
    link_depth: int = 1


@dataclass
class RetrievalResult:
    """Organized experience returned to the agent."""

    episodes: list[Episode] = field(default_factory=list)
    facts: list[GeneralizedFact] = field(default_factory=list)
    scores: list[float] = field(default_factory=list)
    causal_narrative: str = ""
    conflicts: list[str] = field(default_factory=list)


@dataclass
class DecisionResult:
    """Agent investigation decision — returned by decide()."""

    decision: str = ""
    reasoning: str = ""
    context_used: str = ""

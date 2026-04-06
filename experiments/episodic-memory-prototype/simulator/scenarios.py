"""Synthetic scenario generator.

Contains a scripted 9-round debugging scenario designed to test whether
episodic memory can:
  1. Extract and surface a recurring pattern (connection pool exhaustion)
  2. NOT blindly apply that pattern to unrelated failures (red herring)
  3. Generalize the pattern to novel symptoms (subtle)
  4. Learn from corrections and update its hypothesis (correction test)
"""

from __future__ import annotations

from dataclasses import dataclass, field


# ---------------------------------------------------------------------------
# Data structures
# ---------------------------------------------------------------------------

@dataclass
class DebugRound:
    round_num: int
    service: str
    bug_report: str
    investigation_notes: list[str]
    attempted_fix: str
    actual_root_cause: str
    outcome: str                               # "success" | "failure"
    assumption: str = ""
    correction: str = ""
    state_label: str = "debugging"
    is_test_round: bool = False
    expected_keywords: list[str] = field(default_factory=list)

    correct_keywords: list[str] = field(default_factory=list)
    incorrect_keywords: list[str] = field(default_factory=list)
    round_label: str = ""
    trigger_fact_extraction: bool = False


# ---------------------------------------------------------------------------
# Keyword groups reused across rounds
# ---------------------------------------------------------------------------

_POOL_KW = ["connection pool", "pool exhaust", "pool saturat", "pool starvation"]
_DEPLOY_KW = ["deployment", "migration", "config", "deploy", "release"]


# ---------------------------------------------------------------------------
# The 9-round debugging scenario
# ---------------------------------------------------------------------------

def debugging_scenario() -> list[DebugRound]:
    return [
        # -- Round 1: API service, initial encounter -----------------------
        DebugRound(
            round_num=1,
            service="api-gateway",
            bug_report="API returning 500 errors under load.",
            investigation_notes=[
                "Checked application logs: seeing 'connection refused' errors.",
                "Database CPU looks normal, queries are fine.",
                "Noticed Redis client throwing 'pool exhausted' exceptions.",
            ],
            attempted_fix="Increased database connection pool size.",
            actual_root_cause="Redis connection pool exhaustion under high QPS.",
            outcome="failure",
            assumption="database connections",
            correction="Was actually Redis connection pool exhaustion, not DB.",
            correct_keywords=_POOL_KW,
        ),

        # -- Round 2: Payment service, different domain, same pattern ------
        DebugRound(
            round_num=2,
            service="payment-service",
            bug_report="Timeouts in payment processing during peak hours.",
            investigation_notes=[
                "Payment queue depth spiking during peak traffic.",
                "Downstream calls to fraud-check timing out.",
                "Connection pool metrics show 100% utilization on cache layer.",
            ],
            attempted_fix="Added retry logic with exponential backoff.",
            actual_root_cause="Cache connection pool saturated under peak load.",
            outcome="failure",
            assumption="network latency to fraud-check service",
            correction="Connection pool to cache was the bottleneck, not network.",
            correct_keywords=_POOL_KW,
        ),

        # -- Round 3: RED HERRING — looks similar but is NOT pool ----------
        DebugRound(
            round_num=3,
            service="payment-service",
            bug_report=(
                "Payment service returning 504 errors after deployment."
            ),
            investigation_notes=[
                "504s started exactly when v2.14.3 was deployed.",
                "Upstream dependency health checks all passing.",
                "Connection pool metrics look normal — no exhaustion.",
                "Found misconfigured timeout value in new config: "
                "1ms instead of 1000ms.",
            ],
            attempted_fix="Rolled back deployment to v2.14.2.",
            actual_root_cause=(
                "Bad config change in v2.14.3 — upstream timeout set to "
                "1ms instead of 1000ms."
            ),
            outcome="success",
            assumption="",
            correction="",
            is_test_round=True,
            round_label="RED HERRING",
            expected_keywords=[
                "config", "deployment", "rollback", "timeout setting",
            ],
            correct_keywords=[
                "config", "deployment", "rollback", "misconfigur",
                "bad deploy", "timeout setting",
            ],
            incorrect_keywords=["connection pool", "pool exhaust"],
        ),

        # -- Round 4: Search service, pattern solidifies -------------------
        DebugRound(
            round_num=4,
            service="search-service",
            bug_report="Search service 503s during batch indexing.",
            investigation_notes=[
                "Batch indexing triggers 10x normal query volume.",
                "Elasticsearch is healthy, but the service can't reach it.",
                "Thread dump shows all threads blocked waiting for connections.",
            ],
            attempted_fix="Scaled up search service replicas.",
            actual_root_cause=(
                "Connection pool to Elasticsearch exhausted during "
                "indexing spike."
            ),
            outcome="failure",
            assumption="insufficient compute capacity",
            correction=(
                "Connection pool starvation, not compute — need bigger pool."
            ),
            correct_keywords=_POOL_KW,
            trigger_fact_extraction=True,
        ),

        # -- Round 5: Test — does agent recall the pattern? ----------------
        DebugRound(
            round_num=5,
            service="notification-service",
            bug_report="Notification service dropping messages under load.",
            investigation_notes=[
                "Message broker consumer lag increasing rapidly.",
                "Service health checks passing but throughput collapsed.",
            ],
            attempted_fix="",
            actual_root_cause="Connection pool to message broker exhausted.",
            outcome="failure",
            is_test_round=True,
            round_label="TEST",
            expected_keywords=[
                "connection pool", "exhaustion", "under load", "pool",
            ],
            correct_keywords=_POOL_KW,
        ),

        # -- Round 6: Test — does generalized fact surface? ----------------
        DebugRound(
            round_num=6,
            service="auth-service",
            bug_report="Auth service slow during login spike.",
            investigation_notes=[
                "Login latency p99 jumped from 200ms to 8s.",
                "Token validation calls to Redis backing up.",
            ],
            attempted_fix="",
            actual_root_cause=(
                "Redis connection pool exhaustion during auth spike."
            ),
            outcome="failure",
            is_test_round=True,
            round_label="TEST",
            expected_keywords=[
                "connection pool", "Redis", "pool exhaustion", "under load",
            ],
            correct_keywords=_POOL_KW,
        ),

        # -- Round 7: SUBTLE — same root cause, totally different symptoms -
        DebugRound(
            round_num=7,
            service="auth-service",
            bug_report=(
                "Auth service latency P99 increased from 50ms to 200ms, "
                "no errors."
            ),
            investigation_notes=[
                "No 5xx errors, no timeouts — just gradual latency increase.",
                "CPU and memory utilization steady at 40%.",
                "Thread pool looks healthy, no blocked threads.",
            ],
            attempted_fix="",
            actual_root_cause=(
                "Connection pool contention — pool sized for average load "
                "but insufficient during traffic bursts, causing queueing "
                "delays without hard failures."
            ),
            outcome="failure",
            is_test_round=True,
            round_label="SUBTLE",
            expected_keywords=[
                "connection pool", "pool contention", "pool", "queueing",
            ],
            correct_keywords=_POOL_KW + ["pool contention"],
        ),

        # -- Round 8: CORRECTION — agent gets it wrong, learns from it ----
        DebugRound(
            round_num=8,
            service="order-service",
            bug_report=(
                "Order service returning 500 errors after deploying v2.4.1. "
                "Errors started exactly at deploy time, not during a "
                "traffic spike."
            ),
            investigation_notes=[
                "Errors started exactly at deploy time, not during load.",
                "Connection pool metrics look healthy — no saturation.",
                "Database query latency spiked: full table scans detected.",
                "Found that v2.4.1 migration dropped an index on the "
                "orders table.",
            ],
            attempted_fix="Investigated connection pool metrics — all healthy.",
            actual_root_cause=(
                "Missing database index from v2.4.1 migration caused "
                "full table scans and 500 errors."
            ),
            outcome="failure",
            assumption="connection pool exhaustion",
            correction=(
                "Not connection pool — errors correlated with deployment "
                "time, not load. Root cause was missing DB index from "
                "migration."
            ),
            round_label="CORRECTION",
            expected_keywords=["deployment", "migration", "index", "deploy",
                               "config"],
            correct_keywords=_DEPLOY_KW + ["index", "database migration"],
            incorrect_keywords=["connection pool", "pool exhaust"],
            trigger_fact_extraction=True,
        ),

        # -- Round 9: CORRECTION TEST — did the agent learn? --------------
        DebugRound(
            round_num=9,
            service="inventory-service",
            bug_report=(
                "Inventory service errors spiking after v3.1.0 release. "
                "Started immediately after deployment, no change in traffic."
            ),
            investigation_notes=[
                "Error rate jumped from 0 to 15% at exactly deploy time.",
                "Traffic volume unchanged — this is not a load issue.",
                "Rollback to v3.0.9 resolved errors immediately.",
            ],
            attempted_fix="",
            actual_root_cause=(
                "Bad deployment — v3.1.0 introduced a schema mismatch "
                "that caused query failures."
            ),
            outcome="failure",
            is_test_round=True,
            round_label="CORRECTION",
            expected_keywords=["deployment", "migration", "config",
                               "release", "deploy"],
            correct_keywords=_DEPLOY_KW,
            incorrect_keywords=["connection pool", "pool exhaust"],
        ),
    ]

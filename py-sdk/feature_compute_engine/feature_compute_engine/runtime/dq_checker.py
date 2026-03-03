"""
Data quality checks executed after asset computation.
DQ must pass before artifact is published.
"""
from __future__ import annotations

import logging
import operator
import re
from typing import Any, Dict, List

logger = logging.getLogger(__name__)

_OPERATORS = {
    ">": operator.gt,
    ">=": operator.ge,
    "<": operator.lt,
    "<=": operator.le,
    "==": operator.eq,
}

_ROW_COUNT_RE = re.compile(r"row_count\s*(>|>=|<|<=|==)\s*(\d+)")
_NULL_RATE_RE = re.compile(r"null_rate\((\w+)\)\s*(>|>=|<|<=|==)\s*([\d.]+)")
_DISTINCT_COUNT_RE = re.compile(r"distinct_count\((\w+)\)\s*(>|>=|<|<=|==)\s*(\d+)")


class DQChecker:
    """Runs DQ check expressions against a computed DataFrame."""

    def run_checks(
        self, df: Any, checks: List[str], engine: str = "spark"
    ) -> Dict[str, bool]:
        """
        Run each check expression against the DataFrame.
        Returns: ``{check_expression: True/False}``

        Supported check expressions::

            "row_count > 0"
            "null_rate(user_id) < 0.001"
            "distinct_count(user_id) > 100"
        """
        results: Dict[str, bool] = {}
        for check in checks:
            try:
                passed = self._evaluate_check(df, check, engine)
                results[check] = passed
                if not passed:
                    logger.warning("DQ check FAILED: %s", check)
                else:
                    logger.info("DQ check passed: %s", check)
            except Exception:
                logger.error("DQ check ERROR: %s", check, exc_info=True)
                results[check] = False
        return results

    def all_passed(self, results: Dict[str, bool]) -> bool:
        return all(results.values()) if results else True

    def _evaluate_check(self, df: Any, check: str, engine: str) -> bool:
        """Parse and evaluate a single check expression."""
        if "row_count" in check:
            return self._check_row_count(df, check, engine)
        if "null_rate" in check:
            return self._check_null_rate(df, check, engine)
        if "distinct_count" in check:
            return self._check_distinct_count(df, check, engine)

        logger.warning("Unknown check type: %s, skipping", check)
        return True

    def _check_row_count(self, df: Any, check: str, engine: str) -> bool:
        match = _ROW_COUNT_RE.match(check)
        if not match:
            raise ValueError(f"Cannot parse row_count check: {check}")
        op_str, threshold = match.group(1), int(match.group(2))
        count = df.count() if engine == "spark" else len(df)
        return _OPERATORS[op_str](count, threshold)

    def _check_null_rate(self, df: Any, check: str, engine: str) -> bool:
        match = _NULL_RATE_RE.match(check)
        if not match:
            raise ValueError(f"Cannot parse null_rate check: {check}")
        col_name, op_str, threshold = (
            match.group(1),
            match.group(2),
            float(match.group(3)),
        )
        if engine == "spark":
            total = df.count()
            nulls = df.filter(df[col_name].isNull()).count()
        else:
            total = len(df)
            nulls = df[col_name].null_count()
        rate = nulls / total if total > 0 else 0.0
        return _OPERATORS[op_str](rate, threshold)

    def _check_distinct_count(self, df: Any, check: str, engine: str) -> bool:
        match = _DISTINCT_COUNT_RE.match(check)
        if not match:
            raise ValueError(f"Cannot parse distinct_count check: {check}")
        col_name, op_str, threshold = (
            match.group(1),
            match.group(2),
            int(match.group(3)),
        )
        if engine == "spark":
            distinct = df.select(col_name).distinct().count()
        else:
            distinct = df[col_name].n_unique()
        return _OPERATORS[op_str](distinct, threshold)

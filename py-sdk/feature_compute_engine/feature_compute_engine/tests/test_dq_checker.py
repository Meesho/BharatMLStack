"""Tests for DQChecker — uses plain Python mocks, no Spark/Polars."""
from __future__ import annotations

from unittest.mock import MagicMock

import pytest

from feature_compute_engine.runtime.dq_checker import DQChecker


def _mock_df(row_count: int = 100, null_count: int = 0, distinct: int = 50):
    """Build a mock DataFrame with .count(), .filter(), .select() for Spark."""
    df = MagicMock(name="DataFrame")
    df.count.return_value = row_count

    filtered_df = MagicMock()
    filtered_df.count.return_value = null_count
    df.filter.return_value = filtered_df

    distinct_df = MagicMock()
    distinct_df.count.return_value = distinct
    select_df = MagicMock()
    select_df.distinct.return_value = distinct_df
    df.select.return_value = select_df

    return df


class TestRowCountCheck:
    def test_passes_for_nonempty(self) -> None:
        checker = DQChecker()
        results = checker.run_checks(_mock_df(row_count=10), ["row_count > 0"], "spark")
        assert results["row_count > 0"] is True

    def test_fails_for_empty(self) -> None:
        checker = DQChecker()
        results = checker.run_checks(_mock_df(row_count=0), ["row_count > 0"], "spark")
        assert results["row_count > 0"] is False

    def test_greater_equal(self) -> None:
        checker = DQChecker()
        results = checker.run_checks(_mock_df(row_count=5), ["row_count >= 5"], "spark")
        assert results["row_count >= 5"] is True

    def test_less_than(self) -> None:
        checker = DQChecker()
        results = checker.run_checks(
            _mock_df(row_count=1000), ["row_count < 500"], "spark"
        )
        assert results["row_count < 500"] is False

    def test_equality(self) -> None:
        checker = DQChecker()
        results = checker.run_checks(
            _mock_df(row_count=42), ["row_count == 42"], "spark"
        )
        assert results["row_count == 42"] is True


class TestNullRateCheck:
    def test_passes_with_clean_data(self) -> None:
        checker = DQChecker()
        df = _mock_df(row_count=1000, null_count=0)
        results = checker.run_checks(df, ["null_rate(user_id) < 0.01"], "spark")
        assert results["null_rate(user_id) < 0.01"] is True

    def test_fails_with_high_null_rate(self) -> None:
        checker = DQChecker()
        df = _mock_df(row_count=100, null_count=50)
        results = checker.run_checks(df, ["null_rate(user_id) < 0.01"], "spark")
        assert results["null_rate(user_id) < 0.01"] is False

    def test_zero_rows_rate_is_zero(self) -> None:
        checker = DQChecker()
        df = _mock_df(row_count=0, null_count=0)
        results = checker.run_checks(df, ["null_rate(user_id) < 0.01"], "spark")
        assert results["null_rate(user_id) < 0.01"] is True


class TestDistinctCountCheck:
    def test_passes(self) -> None:
        checker = DQChecker()
        df = _mock_df(distinct=200)
        results = checker.run_checks(
            df, ["distinct_count(user_id) > 100"], "spark"
        )
        assert results["distinct_count(user_id) > 100"] is True

    def test_fails(self) -> None:
        checker = DQChecker()
        df = _mock_df(distinct=50)
        results = checker.run_checks(
            df, ["distinct_count(user_id) > 100"], "spark"
        )
        assert results["distinct_count(user_id) > 100"] is False


class TestUnknownCheck:
    def test_unknown_check_passes_with_warning(self) -> None:
        checker = DQChecker()
        df = _mock_df()
        results = checker.run_checks(df, ["some_unknown_check > 5"], "spark")
        assert results["some_unknown_check > 5"] is True


class TestAllPassed:
    def test_all_passed_true(self) -> None:
        checker = DQChecker()
        assert checker.all_passed({"a": True, "b": True}) is True

    def test_all_passed_false(self) -> None:
        checker = DQChecker()
        assert checker.all_passed({"a": True, "b": False}) is False

    def test_all_passed_empty(self) -> None:
        checker = DQChecker()
        assert checker.all_passed({}) is True


class TestMultipleChecks:
    def test_multiple_checks_mixed(self) -> None:
        checker = DQChecker()
        df = _mock_df(row_count=100, null_count=0, distinct=200)
        results = checker.run_checks(
            df,
            ["row_count > 0", "null_rate(user_id) < 0.01", "distinct_count(user_id) > 100"],
            "spark",
        )
        assert len(results) == 3
        assert all(results.values())

    def test_malformed_check_counts_as_failure(self) -> None:
        checker = DQChecker()
        df = _mock_df()
        results = checker.run_checks(df, ["row_count BADOP 0"], "spark")
        assert results["row_count BADOP 0"] is False

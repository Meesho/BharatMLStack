"""Tests for InputReader — uses plain Python mocks, no Spark/Polars."""
from __future__ import annotations

from unittest.mock import MagicMock, patch

from feature_compute_engine.runtime.input_reader import InputReader


class TestInputReaderCache:
    def test_cache_hit_returns_same_object(self) -> None:
        """Second read with same args returns cached DataFrame."""
        mock_df = MagicMock(name="DataFrame")
        reader = InputReader(engine="spark")

        with patch.object(reader, "_read_spark", return_value=mock_df) as mock_read:
            df1 = reader.read("orders", "/path/to/orders", "2026-03-01")
            df2 = reader.read("orders", "/path/to/orders", "2026-03-01")

        assert df1 is df2
        mock_read.assert_called_once()

    def test_different_partitions_are_separate_cache_entries(self) -> None:
        mock_df1 = MagicMock(name="df_march")
        mock_df2 = MagicMock(name="df_april")
        reader = InputReader(engine="spark")

        with patch.object(reader, "_read_spark", side_effect=[mock_df1, mock_df2]):
            df1 = reader.read("orders", "/path", "2026-03-01")
            df2 = reader.read("orders", "/path", "2026-04-01")

        assert df1 is not df2
        assert df1 is mock_df1
        assert df2 is mock_df2


class TestInputReaderInjectOutput:
    def test_inject_output_available_via_cache(self) -> None:
        reader = InputReader(engine="spark")
        mock_df = MagicMock(name="injected_df")

        reader.inject_output("fg.user_spend", mock_df)

        assert reader._cache["fg.user_spend"] is mock_df

    def test_injected_output_not_read_from_storage(self) -> None:
        """After inject_output, read() returns the injected DF directly."""
        reader = InputReader(engine="spark")
        mock_df = MagicMock(name="injected_df")

        reader.inject_output("fg.user_spend", mock_df)

        with patch.object(reader, "_read_spark") as mock_read:
            result = reader.read("fg.user_spend", "fg.user_spend")
            mock_read.assert_not_called()

        assert result is mock_df


class TestInputReaderReadShared:
    def test_read_shared_reads_all_inputs(self) -> None:
        reader = InputReader(engine="spark")
        mock_df_a = MagicMock(name="df_a")
        mock_df_b = MagicMock(name="df_b")

        with patch.object(reader, "_read_spark", side_effect=[mock_df_a, mock_df_b]):
            result = reader.read_shared({
                "source_a": "/path/a",
                "source_b": "/path/b",
            })

        assert result["source_a"] is mock_df_a
        assert result["source_b"] is mock_df_b

    def test_read_shared_empty(self) -> None:
        reader = InputReader(engine="spark")
        result = reader.read_shared({})
        assert result == {}


class TestInputReaderUnsupportedEngine:
    def test_unsupported_engine_raises(self) -> None:
        reader = InputReader(engine="duckdb")
        try:
            reader.read("orders", "/path", None)
            assert False, "Should have raised ValueError"
        except ValueError as e:
            assert "Unsupported engine" in str(e)

"""
Integration tests for the proto decode optimization pipeline.

Uses fixtures: tests/fixtures/schema_120.json (120 features including score, pctr_score, pcvr_score).
Metadata byte "BA==" decodes to version/format; feature types include DataTypeInt64, DataTypeFP32,
DataTypeInt32, and FP32 (inconsistent naming for pctr_score, pcvr_score).
"""

import base64
import json
import struct
from pathlib import Path
from unittest.mock import patch

import pytest

from inference_logging_client.decode_plan import (
    compile_selective_plan,
    normalize_feature_type,
    try_build_fixed_plan,
)
from inference_logging_client.proto_decoder import decode_proto_fixed, decode_proto_selective
from inference_logging_client.types import FeatureInfo

FIXTURES_DIR = Path(__file__).resolve().parent / "fixtures"
SCHEMA_120_PATH = FIXTURES_DIR / "schema_120.json"


def _load_schema_120():
    """Load 120-feature schema from fixture JSON (API-style: data[].feature_name, feature_type)."""
    with open(SCHEMA_120_PATH) as f:
        data = json.load(f)
    return [
        FeatureInfo(
            name=c["feature_name"],
            feature_type=c["feature_type"].upper(),
            index=idx,
        )
        for idx, c in enumerate(data["data"])
    ]


def _build_entity_proto_bytes(schema, *, score_val=12345, pctr_val=1.5, pcvr_val=0.25):
    """Build one entity's proto bytes: 1-byte flag + features in schema order (all scalars)."""
    from inference_logging_client.utils import get_scalar_size

    buf = bytearray([1])  # generated flag
    for f in schema:
        sz = get_scalar_size(f.feature_type)
        if sz is None:
            raise ValueError(f"Unsupported type for fixture: {f.feature_type}")
        if f.name == "score":
            buf.extend(struct.pack("<q", score_val))
        elif f.name == "pctr_score":
            buf.extend(struct.pack("<f", pctr_val))
        elif f.name == "pcvr_score":
            buf.extend(struct.pack("<f", pcvr_val))
        else:
            buf.extend(b"\x00" * sz)
    return bytes(buf)


def _build_features_json_188_entities(schema, *, score_val=12345, pctr_val=1.5, pcvr_val=0.25):
    """Build JSON array of 188 entities, each with encoded_features (base64 proto)."""
    one_entity = _build_entity_proto_bytes(schema, score_val=score_val, pctr_val=pctr_val, pcvr_val=pcvr_val)
    b64 = base64.b64encode(one_entity).decode("ascii")
    entities = [{"encoded_features": b64} for _ in range(188)]
    return json.dumps(entities)


# --- 1. Test compile_skip_plan / compile_selective_plan ---


class TestCompilePlan:
    """Test compile_selective_plan with real 120-feature schema."""

    def test_compile_plan_all_columns(self):
        schema = _load_schema_120()
        assert len(schema) == 120
        plan = compile_selective_plan(schema, needed_columns=None)
        decode_entries = [e for e in plan if e[0] in ("scalar", "var") and e[4] is True]
        assert len(decode_entries) == 120
        names = [e[1] for e in decode_entries]
        assert "score" in names
        assert "pctr_score" in names
        assert "pcvr_score" in names
        assert schema[0].feature_type.upper() == "DATATYPEINT64"
        assert schema[-2].name == "pctr_score" and schema[-1].name == "pcvr_score"

    def test_compile_plan_selective_three_columns(self):
        schema = _load_schema_120()
        plan = compile_selective_plan(schema, needed_columns={"score", "pctr_score", "pcvr_score"})
        decode_entries = [e for e in plan if e[0] in ("scalar", "var") and e[4] is True]
        assert len(decode_entries) == 3
        names = {e[1] for e in decode_entries}
        assert names == {"score", "pctr_score", "pcvr_score"}
        skip_bytes_entries = [e for e in plan if e[0] == "skip_bytes"]
        assert len(skip_bytes_entries) >= 1
        total_skipped = sum(e[1] for e in skip_bytes_entries)
        assert total_skipped > 0

    def test_contiguous_scalar_runs_collapsed(self):
        schema = [
            FeatureInfo("a", "INT32", 0),
            FeatureInfo("b", "INT32", 1),
            FeatureInfo("c", "INT32", 2),
            FeatureInfo("wanted", "FP32", 3),
        ]
        plan = compile_selective_plan(schema, needed_columns={"wanted"})
        skip_bytes = [e for e in plan if e[0] == "skip_bytes"]
        assert len(skip_bytes) == 1
        assert skip_bytes[0][1] == 12  # 3 * 4 bytes

    def test_data_type_fp32_and_fp32_both_handled(self):
        schema = _load_schema_120()
        plan = compile_selective_plan(schema, needed_columns=None)
        pctr = next(e for e in plan if e[0] == "scalar" and e[1] == "pctr_score")
        pcvr = next(e for e in plan if e[0] == "scalar" and e[1] == "pcvr_score")
        assert pctr[5] == "FP32"  # canonical type
        assert pcvr[5] == "FP32"
        assert pctr[2] == 4 and pcvr[2] == 4  # fixed_size


# --- 2. Test decode_proto_selective ---


class TestDecodeProtoSelective:
    """Test decode_proto_selective with real encoded bytes."""

    def test_decode_full_plan(self):
        schema = _load_schema_120()
        plan = compile_selective_plan(schema, needed_columns=None)
        entity_bytes = _build_entity_proto_bytes(schema, score_val=999, pctr_val=2.5, pcvr_val=0.1)
        result = decode_proto_selective(entity_bytes, plan)
        assert len(result) == 120
        assert result["score"] == 999
        assert abs(result["pctr_score"] - 2.5) < 1e-5
        assert abs(result["pcvr_score"] - 0.1) < 1e-5
        for k, v in result.items():
            assert v is not None or k not in ("score", "pctr_score", "pcvr_score")

    def test_decode_selective_three_columns(self):
        schema = _load_schema_120()
        plan = compile_selective_plan(schema, needed_columns={"score", "pctr_score", "pcvr_score"})
        entity_bytes = _build_entity_proto_bytes(schema)
        result = decode_proto_selective(entity_bytes, plan)
        assert set(result.keys()) == {"score", "pctr_score", "pcvr_score"}
        assert result["score"] == 12345
        assert abs(result["pctr_score"] - 1.5) < 1e-5
        assert abs(result["pcvr_score"] - 0.25) < 1e-5

    def test_decoded_value_types(self):
        schema = _load_schema_120()
        plan = compile_selective_plan(schema, needed_columns={"score", "pctr_score", "pcvr_score", "f1", "f2"})
        entity_bytes = _build_entity_proto_bytes(schema)
        result = decode_proto_selective(entity_bytes, plan)
        assert isinstance(result["score"], int)
        assert isinstance(result["pctr_score"], float)
        assert isinstance(result["pcvr_score"], float)
        assert isinstance(result["f1"], int)
        assert isinstance(result["f2"], float)


# --- 3. Test end-to-end with SparkSession ---


class TestDecodePipelineE2E:
    """E2E tests with local SparkSession and mocked get_feature_schema (uses spark from conftest)."""

    @pytest.fixture
    def schema_120(self):
        return _load_schema_120()

    @pytest.fixture
    def sample_row_data(self, schema_120):
        features_json = _build_features_json_188_entities(schema_120)
        entities_json = json.dumps([str(i) for i in range(188)])
        return {
            "mp_config_id": "test-config",
            "metadata": "BA==",
            "features": features_json,
            "entities": entities_json,
        }

    def test_decode_single_config_stringify(self, spark, schema_120, sample_row_data):
        from inference_logging_client import api

        with patch("inference_logging_client.plan_builder.get_feature_schema", return_value=schema_120):
            df = spark.createDataFrame([sample_row_data])
            result = api.decode_single_config(
                df, spark, "test-config",
                stringify_features=True,
            )
            assert result.count() == 188
            cols = result.columns
            assert "entity_id" in cols
            assert "score" in cols
            assert "pctr_score" in cols
            assert "pcvr_score" in cols
            metadata_and_system = {"entity_id", "mp_config_id", "metadata", "features", "entities"}
            metadata_and_system.update(api.ROW_METADATA_COLUMNS)
            feature_cols = [c for c in cols if c not in metadata_and_system]
            assert len(feature_cols) >= 120
            row = result.first()
            assert str(row["score"]) == "12345"

    def test_decode_single_config_typed_and_type_map(self, spark, schema_120, sample_row_data):
        from inference_logging_client import api
        from pyspark.sql.types import FloatType

        with patch("inference_logging_client.plan_builder.get_feature_schema", return_value=schema_120):
            df = spark.createDataFrame([sample_row_data])
            result, type_map = api.decode_single_config(
                df, spark, "test-config",
                stringify_features=False,
            )
            assert isinstance(type_map, dict)
            assert type_map.get("pctr_score") == "float"
            assert type_map.get("score") == "int64"
            assert result.schema["pctr_score"].dataType == FloatType()

    def test_decode_single_config_needed_columns(self, spark, schema_120, sample_row_data):
        from inference_logging_client import api

        with patch("inference_logging_client.plan_builder.get_feature_schema", return_value=schema_120):
            df = spark.createDataFrame([sample_row_data])
            result = api.decode_single_config(
                df, spark, "test-config",
                needed_columns={"score"},
                stringify_features=True,
            )
            assert result.count() == 188
            metadata_and_system = {"entity_id", "mp_config_id", "metadata", "features", "entities"}
            metadata_and_system.update(api.ROW_METADATA_COLUMNS)
            feature_cols = [c for c in result.columns if c not in metadata_and_system]
            assert feature_cols == ["score"]


# --- 4. Test from_json fallback ---


class TestFromJsonFallback:
    """Verify from_json yields null for malformed JSON; UDF skips row when Python fallback also fails."""

    def test_from_json_produces_null_for_malformed_features(self, spark):
        from inference_logging_client.spark_utils import add_version_column, parse_json_columns
        from pyspark.sql import Row

        df = spark.createDataFrame([
            Row(mp_config_id="x", metadata="BA==", features="not valid json [", entities="[]"),
        ])
        df = parse_json_columns(df, features_column="features", entities_column="entities")
        df = add_version_column(df, metadata_column="metadata")
        assert df.filter("features_parsed is null").count() >= 1

    def test_udf_skips_row_when_both_from_json_and_python_loads_fail(self, spark, schema_120):
        """One row with malformed features: JVM from_json -> null; Python json.loads fails; UDF skips row."""
        from inference_logging_client import api
        from pyspark.sql import Row

        row = Row(
            mp_config_id="test-config",
            metadata="BA==",
            features="not valid json [",
            entities="[]",
        )
        df = spark.createDataFrame([row])
        with patch("inference_logging_client.plan_builder.get_feature_schema", return_value=schema_120):
            result = api.decode_single_config(df, spark, "test-config", stringify_features=True)
        assert result.count() == 0


# --- 5. Test type normalization ---


class TestTypeNormalization:
    """FP32 vs DataTypeFP32; unknown raises."""

    def test_fp32_and_data_type_fp32_same_plan_entry(self):
        s1 = [FeatureInfo("a", "FP32", 0)]
        s2 = [FeatureInfo("a", "DataTypeFP32", 0)]
        p1 = compile_selective_plan(s1, needed_columns=None)
        p2 = compile_selective_plan(s2, needed_columns=None)
        assert p1[0][5] == "FP32"
        assert p2[0][5] == "FP32"
        assert p1[0][2] == p2[0][2] == 4

    def test_unknown_type_raises(self):
        with pytest.raises(ValueError, match="Unknown feature type"):
            normalize_feature_type("UnknownType")
        with pytest.raises(ValueError, match="Unknown feature type"):
            normalize_feature_type("DataTypeUnknown")

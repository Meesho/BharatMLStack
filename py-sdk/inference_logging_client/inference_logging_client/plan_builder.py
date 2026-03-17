"""Driver-side: fetch schemas, compile decode plans, broadcast to executors; build output schema."""

from __future__ import annotations

import warnings
from typing import Any, Dict, List, Optional, Set, Tuple

from .decode_plan import compile_selective_plan, normalize_feature_type, try_build_fixed_plan
from .io import get_feature_schema
from .types import FeatureInfo

# Canonical feature type (from normalize_feature_type) -> simple type string for type_map / Spark
_CANONICAL_TO_TYPE_STR: Dict[str, str] = {
    "INT8": "int8",
    "I8": "int8",
    "INT16": "int16",
    "I16": "int16",
    "SHORT": "int16",
    "INT32": "int32",
    "I32": "int32",
    "INT": "int32",
    "INT64": "int64",
    "I64": "int64",
    "LONG": "int64",
    "UINT8": "uint8",
    "U8": "uint8",
    "UINT16": "uint16",
    "U16": "uint16",
    "UINT32": "uint32",
    "U32": "uint32",
    "UINT64": "uint64",
    "U64": "uint64",
    "FP8E5M2": "float",
    "FP8E4M3": "float",
    "FP16": "float",
    "FLOAT16": "float",
    "F16": "float",
    "FP32": "float",
    "FLOAT32": "float",
    "F32": "float",
    "FLOAT": "float",
    "FP64": "double",
    "FLOAT64": "double",
    "F64": "double",
    "DOUBLE": "double",
    "BOOL": "boolean",
    "BOOLEAN": "boolean",
    "STRING": "string",
    "STR": "string",
    "BYTES": "binary",
}
# Vector types: treat as string for output schema when not stringify
for _v in (
    "FP8E5M2VECTOR", "FP8E4M3VECTOR", "FP16VECTOR", "FP32VECTOR", "FP64VECTOR",
    "INT8VECTOR", "INT16VECTOR", "INT32VECTOR", "INT64VECTOR",
    "UINT8VECTOR", "UINT16VECTOR", "UINT32VECTOR", "UINT64VECTOR",
    "STRINGVECTOR", "BOOLVECTOR",
):
    _CANONICAL_TO_TYPE_STR[_v] = "string"


def _output_names_from_plan(plan_kind: str, plan_value: Any, schema: List[FeatureInfo]) -> Set[str]:
    """Extract set of feature names that will be decoded (should_decode=True)."""
    out: Set[str] = set()
    if plan_kind == "fixed":
        # fixed_plan = (offsets, sizes, names, types)
        out.update(plan_value[2])
    else:
        # general: selective plan entries ("scalar", name, ...) or ("var", name, ...) with entry[4] True
        for entry in plan_value:
            if entry[0] in ("scalar", "var") and entry[4] is True:
                out.add(entry[1])
    return out


def _type_map_from_schemas(
    plans: Dict[Tuple[str, int], Tuple[str, Any, List[FeatureInfo]]],
    output_feature_names: Set[str],
) -> Dict[str, str]:
    """Build feature_name -> type string from all schemas in plans."""
    name_to_canonical: Dict[str, str] = {}
    for _key, (_kind, _plan, schema) in plans.items():
        for f in schema:
            if f.name not in output_feature_names:
                continue
            try:
                canonical = normalize_feature_type(f.feature_type)
                name_to_canonical[f.name] = canonical
            except ValueError:
                continue
    return {
        name: _CANONICAL_TO_TYPE_STR.get(canonical, "string")
        for name, canonical in name_to_canonical.items()
    }


def build_and_broadcast_plans(
    spark: Any,
    distinct_pairs: List[Tuple[str, int]],
    inference_host: str,
    needed_columns: Optional[Set[str]] = None,
    stringify_features: bool = True,
) -> Tuple[Any, Set[str], Dict[str, str]]:
    """Fetch schemas for all (config, version), compile plans, broadcast. Driver-only.

    Returns:
        broadcast_plans: Broadcast of dict[(config_id, version)] -> ("fixed"|"general", plan, schema)
        output_feature_names: set of feature names that will be decoded
        type_map: feature_name -> type string (empty if stringify_features=True)
    """
    plans: Dict[Tuple[str, int], Tuple[str, Any, List[FeatureInfo]]] = {}
    output_feature_names: Set[str] = set()

    for config_id, version in distinct_pairs:
        try:
            schema = get_feature_schema(config_id, version, inference_host)
        except Exception as e:
            warnings.warn(
                f"Failed to fetch schema for (config_id={config_id!r}, version={version}): {e}",
                UserWarning,
            )
            continue
        if not schema:
            continue
        try:
            selective_plan = compile_selective_plan(schema, needed_columns=needed_columns)
        except ValueError as e:
            warnings.warn(
                f"Failed to compile selective plan for ({config_id!r}, {version}): {e}",
                UserWarning,
            )
            continue
        fixed_plan = try_build_fixed_plan(schema)
        if fixed_plan is not None:
            plans[(config_id, version)] = ("fixed", fixed_plan, schema)
        else:
            plans[(config_id, version)] = ("general", selective_plan, schema)
        key = (config_id, version)
        kind, plan_val, sch = plans[key]
        output_feature_names.update(_output_names_from_plan(kind, plan_val, sch))

    type_map: Dict[str, str] = {}
    if not stringify_features and plans:
        type_map = _type_map_from_schemas(plans, output_feature_names)

    broadcast_plans = spark.sparkContext.broadcast(plans)
    return broadcast_plans, output_feature_names, type_map


def build_output_schema(
    df: Any,
    metadata_cols: List[str],
    output_feature_names: Set[str],
    plans: Dict[Tuple[str, int], Tuple[str, Any, List[FeatureInfo]]],
    stringify_features: bool,
) -> Any:
    """Build StructType for decode output: entity_id, metadata columns (original types), feature columns.

    - entity_id: StringType first
    - metadata_cols: preserve original Spark types from df
    - feature columns: StringType if stringify_features else mapped from feature types in plans
    - Only features in output_feature_names (narrow schema)
    """
    from pyspark.sql.types import (
        BooleanType,
        DoubleType,
        FloatType,
        IntegerType,
        LongType,
        StringType,
        StructField,
        StructType,
    )

    fields: List[StructField] = [StructField("entity_id", StringType(), True)]
    input_field_map = {f.name: f.dataType for f in df.schema.fields}

    for col_name in metadata_cols:
        if col_name not in df.columns:
            continue
        dtype = input_field_map.get(col_name, StringType())
        fields.append(StructField(col_name, dtype, True))

    _type_str_to_spark = {
        "float": FloatType(),
        "double": DoubleType(),
        "int8": IntegerType(),
        "int16": IntegerType(),
        "int32": IntegerType(),
        "int64": LongType(),
        "uint8": IntegerType(),
        "uint16": IntegerType(),
        "uint32": IntegerType(),
        "uint64": LongType(),
        "string": StringType(),
        "boolean": BooleanType(),
        "binary": StringType(),
    }

    if stringify_features:
        for name in sorted(output_feature_names):
            fields.append(StructField(name, StringType(), True))
    else:
        type_map = _type_map_from_schemas(plans, output_feature_names)
        for name in sorted(output_feature_names):
            type_str = type_map.get(name, "string")
            spark_type = _type_str_to_spark.get(type_str, StringType())
            fields.append(StructField(name, spark_type, True))

    return StructType(fields)

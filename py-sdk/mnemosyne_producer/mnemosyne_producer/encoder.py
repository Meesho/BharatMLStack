"""
Row → persist.Data → bytes.

For each entity row we build a persist.Data proto containing:
  * key_values: the entity's key columns as strings (e.g. ["abc123", "99"]).
  * feature_values: one `FeatureValues` per feature-group, in the order
                    declared in the config. Each FG's `Values` holds a single
                    typed parallel array (fp32_values / int32_values / ...) in
                    feature-label order.

Null source values are replaced by the per-feature default declared in the
config (cast to the target type once at encoder-construction time).

We pre-compute as much as possible at __init__ (data-type branches, casters,
default arrays) so the hot path is tight.

Vectors are supported for the existing proto vector types; for V1 we only
exercise scalar paths from the user's geo config.
"""
from __future__ import annotations

from typing import Any, Callable, Dict, List

from .config import Config, FeatureGroupSpec, FeatureSpec


# Group data types by the proto field they populate.
_FP32_TYPES = {"DataTypeFP8E5M2", "DataTypeFP8E4M3", "DataTypeFP16", "DataTypeFP32"}
_FP64_TYPES = {"DataTypeFP64"}
_I32_TYPES = {"DataTypeInt8", "DataTypeInt16", "DataTypeInt32"}
_I64_TYPES = {"DataTypeInt64"}
_U32_TYPES = {"DataTypeUint8", "DataTypeUint16", "DataTypeUint32"}
_U64_TYPES = {"DataTypeUint64"}

_FP32_VEC = {"DataTypeFP8E5M2Vector", "DataTypeFP8E4M3Vector",
             "DataTypeFP16Vector", "DataTypeFP32Vector"}
_FP64_VEC = {"DataTypeFP64Vector"}
_I32_VEC = {"DataTypeInt8Vector", "DataTypeInt16Vector", "DataTypeInt32Vector"}
_I64_VEC = {"DataTypeInt64Vector"}
_U32_VEC = {"DataTypeUint8Vector", "DataTypeUint16Vector", "DataTypeUint32Vector"}
_U64_VEC = {"DataTypeUint64Vector"}


def _cast_default(default_str: str, data_type: str) -> Any:
    """Cast the textual default-value from JSON to the target Python type."""
    if data_type in _FP32_TYPES or data_type in _FP32_VEC:
        return float(default_str or "0")
    if data_type in _FP64_TYPES or data_type in _FP64_VEC:
        return float(default_str or "0")
    if data_type in _I32_TYPES or data_type in _I32_VEC \
            or data_type in _I64_TYPES or data_type in _I64_VEC:
        return int(default_str or "0")
    if data_type in _U32_TYPES or data_type in _U32_VEC \
            or data_type in _U64_TYPES or data_type in _U64_VEC:
        return int(default_str or "0")
    if data_type == "DataTypeString" or data_type == "DataTypeStringVector":
        return default_str or ""
    if data_type == "DataTypeBool" or data_type == "DataTypeBoolVector":
        s = (default_str or "0").strip().lower()
        return s in ("1", "true", "yes", "y", "t")
    raise ValueError(f"unsupported data_type: {data_type}")


class _FgPlan:
    """Pre-built per-feature-group encoding plan."""
    __slots__ = ("label", "data_type", "source_columns", "defaults", "is_vector")

    def __init__(self, fg: FeatureGroupSpec):
        self.label = fg.label
        self.data_type = fg.data_type
        self.is_vector = "Vector" in fg.data_type
        self.source_columns: List[str] = [f.source_column for f in fg.features]
        self.defaults: List[Any] = [_cast_default(f.default_value, f.data_type) for f in fg.features]


class RowEncoder:
    """Builds persist.Data bytes from a row dict."""

    def __init__(self, cfg: Config):
        self.cfg = cfg
        self.fg_plans: List[_FgPlan] = [_FgPlan(fg) for fg in cfg.feature_groups]

    # ---------- public API ----------

    def encode(self, row: Dict[str, Any]) -> bytes:
        # Local import so the proto package is resolved on Spark executors
        # (the import only happens once per partition since RowEncoder is
        # constructed there).
        from bharatml_commons.proto.persist.persist_pb2 import Data, FeatureValues, Values  # noqa: WPS433

        key_values = [
            ("" if row.get(c) is None else str(row.get(c)))
            for c in self.cfg.key_columns
        ]

        feature_values = []
        for plan in self.fg_plans:
            vals = Values()
            self._fill(vals, plan, row)
            feature_values.append(FeatureValues(values=vals))

        return Data(key_values=key_values, feature_values=feature_values).SerializeToString()

    # ---------- internals ----------

    def _fill(self, values: "Any", plan: _FgPlan, row: Dict[str, Any]) -> None:
        dt = plan.data_type
        # Resolve per-feature value or default
        # NOTE: row.get(col) returns None for missing/null; defaults already typed.
        if dt in _FP32_TYPES:
            arr = [self._or_default(row.get(c), d, float) for c, d in zip(plan.source_columns, plan.defaults)]
            values.fp32_values.extend(arr)  # proto field is double[] but holds fp32 logically
        elif dt in _FP64_TYPES:
            arr = [self._or_default(row.get(c), d, float) for c, d in zip(plan.source_columns, plan.defaults)]
            values.fp64_values.extend(arr)
        elif dt in _I32_TYPES:
            arr = [self._or_default(row.get(c), d, int) for c, d in zip(plan.source_columns, plan.defaults)]
            values.int32_values.extend(arr)
        elif dt in _I64_TYPES:
            arr = [self._or_default(row.get(c), d, int) for c, d in zip(plan.source_columns, plan.defaults)]
            values.int64_values.extend(arr)
        elif dt in _U32_TYPES:
            arr = [self._or_default(row.get(c), d, int) for c, d in zip(plan.source_columns, plan.defaults)]
            values.uint32_values.extend(arr)
        elif dt in _U64_TYPES:
            arr = [self._or_default(row.get(c), d, int) for c, d in zip(plan.source_columns, plan.defaults)]
            values.uint64_values.extend(arr)
        elif dt == "DataTypeString":
            arr = [self._or_default(row.get(c), d, str) for c, d in zip(plan.source_columns, plan.defaults)]
            values.string_values.extend(arr)
        elif dt == "DataTypeBool":
            arr = [self._or_default(row.get(c), d, bool) for c, d in zip(plan.source_columns, plan.defaults)]
            values.bool_values.extend(arr)
        elif plan.is_vector:
            self._fill_vector(values, plan, row)
        else:
            raise ValueError(f"unsupported data_type: {dt}")

    def _fill_vector(self, values: "Any", plan: _FgPlan, row: Dict[str, Any]) -> None:
        from bharatml_commons.proto.persist.persist_pb2 import Values, Vector  # noqa: WPS433
        dt = plan.data_type
        for src, default in zip(plan.source_columns, plan.defaults):
            raw = row.get(src)
            seq = raw if raw is not None else default  # default may be scalar; vector default rarely useful
            inner = Values()
            if dt in _FP32_VEC:
                inner.fp32_values.extend(float(x) for x in seq)
            elif dt in _FP64_VEC:
                inner.fp64_values.extend(float(x) for x in seq)
            elif dt in _I32_VEC:
                inner.int32_values.extend(int(x) for x in seq)
            elif dt in _I64_VEC:
                inner.int64_values.extend(int(x) for x in seq)
            elif dt in _U32_VEC:
                inner.uint32_values.extend(int(x) for x in seq)
            elif dt in _U64_VEC:
                inner.uint64_values.extend(int(x) for x in seq)
            elif dt == "DataTypeStringVector":
                inner.string_values.extend(str(x) for x in seq)
            elif dt == "DataTypeBoolVector":
                inner.bool_values.extend(bool(x) for x in seq)
            else:
                raise ValueError(f"unsupported vector data_type: {dt}")
            values.vector.append(Vector(values=inner))

    @staticmethod
    def _or_default(val: Any, default: Any, caster: Callable[[Any], Any]) -> Any:
        if val is None:
            return default
        try:
            # Fast path: already correct type
            return caster(val)
        except (TypeError, ValueError):
            # NaN / inf for floats → default for safety
            return default

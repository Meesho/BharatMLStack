# Predict APIs and Feature Logging

This document describes the Predict gRPC APIs exposed by Inferflow, and the high-level request/logging flow for PointWise, PairWise, and SlateWise inference.

## APIs Exposed

The `Predict` service exposes three RPCs:

- `InferPointWise(PointWiseRequest) returns (PointWiseResponse)`
- `InferPairWise(PairWiseRequest) returns (PairWiseResponse)`
- `InferSlateWise(SlateWiseRequest) returns (SlateWiseResponse)`

Reference: `inferflow/server/proto/predict.proto`.

## API Intent

### PointWise (sometimes referred to as "pintwise")

- Use when you need per-target scoring.
- Input:
  - `context_features` (request-level features)
  - `target_input_schema` (schema for `Target.feature_values`)
  - `targets` (entities/items to score)
- Output:
  - `target_output_schema`
  - `target_scores` (one score row per target)
  - `request_error` (request-level error)

### PairWise

- Use when you need pair-level scoring/ranking and optional per-target outputs.
- Input:
  - `targets` (base target pool)
  - `pairs` (`first_target_index` + `second_target_index`)
  - `pair_input_schema` for pair-level features
  - `target_input_schema` for target-level features
- Output:
  - `pair_scores` aligned with `pair_output_schema`
  - `target_scores` aligned with `target_output_schema`
  - `request_error`

### SlateWise

- Use when you need slate-level scoring plus optional per-target outputs.
- Input:
  - `targets` (base target pool)
  - `slates` (`target_indices` per slate)
  - `slate_input_schema` for slate-level features
  - `target_input_schema` for target-level features
- Output:
  - `slate_scores` aligned with `slate_output_schema`
  - `target_scores` aligned with `target_output_schema`
  - `request_error`

## High-Level Flow

The runtime path is common across all three APIs with adapter-specific shaping:

1. Receive gRPC request in `PredictService` (`predict_handler.go`).
2. Load model config via `config.GetModelConfig(model_config_id)`.
3. Adapt request into `components.ComponentRequest` (`predict_adapter.go`):
   - Build `ComponentData` (target matrix).
   - For PairWise/SlateWise, also build `SlateData` (slate matrix).
4. Execute DAG components via `executor.Execute(...)`.
5. Build RPC response from matrices (`predict_response.go`):
   - PointWise from target matrix.
   - PairWise/SlateWise from target + slate matrices.
6. Emit metrics (`request.total`, `latency`, `batch.size`).
7. Optionally trigger feature logging asynchronously (`maybeLogInferenceResponse`).

## Matrix Model Used by Predict

- `ComponentData`:
  - Main per-target matrix.
  - Always present.
- `SlateData`:
  - Per-slate matrix.
  - Present for PairWise and SlateWise.
  - Contains `slate_target_indices` and slate-level features.

## Feature Logging: Current Behavior

Feature logging is implemented in `inferflow/handlers/inferflow/feature_logging.go`.

### Trigger Conditions

Logging is attempted only when all are true:

- `conf.ResponseConfig.LoggingPerc > 0`
- Random sampling check passes: `rand.Intn(100)+1 <= LoggingPerc`
- `tracking_id` is non-empty

Reference: `maybeLogInferenceResponse` in `predict_handler.go`.

### Logging Format

V2 format is selected using:

- `config.GetModelConfigMap().ServiceConfig.V2LoggingType`

Supported format values:

- `proto`
- `arrow`
- `parquet`

### Logged Message Shape

Logged payload uses `InferflowLog` (`inferflow_logging.proto`):

- `user_id`
- `tracking_id`
- `model_config_id`
- `entities`
- `features` (`PerEntityFeatures.encoded_features`)
- `metadata`
- `parent_entity`

### What Data Is Logged Today

Current logging functions (`logInferflowResponseBytes`, `logInferflowResponseArrow`, `logInferflowResponseParquet`) read from:

- `compRequest.ComponentData` (target matrix)

They do not currently read from:

- `compRequest.SlateData` (slate matrix)

So for PairWise/SlateWise requests, logging currently captures target-matrix features, not slate-matrix rows.

### Metadata and Transport

- Metadata byte packs:
  - compression-enabled bit
  - cache version
  - format type (proto/arrow/parquet)
- Logs are sent to Kafka through prism logger (`PublishInferenceInsightsLog`).
- Event name used: `inferflow_inference_logs`.

### Batching

- Logs are batched before Kafka publish.
- Batch size uses `ResponseConfig.LogBatchSize`, default `500`.

## Notes for Future Slate Logging

If slate logging is required, the cleanest approach is to add a parallel slate logging path that:

- reads from `compRequest.SlateData`
- builds slate-oriented encoded payloads
- publishes with same `tracking_id` and `model_config_id` for correlation

This avoids breaking existing target-log consumers.
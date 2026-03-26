# Inferflow

Inferflow handles inferflow config lifecycle operations: onboarding, review/approval, promote, edit, clone, scale-up, delete, cancel, list (get-all / get-all-requests), validation, functional testing, and feature schema generation.

## Package layout

```
internal/inferflow/
├── controller/   (wires routes to handler)
├── handler/      (Config implementation + helpers)
├── etcd/         (ETCD config read/write for inferflow)
├── route/
├── handler/proto/ and generated Go
├── init.go
└── README.md
```

---

## Handler package structure

| File | Purpose |
|------|--------|
| **config.go** | Defines the **Config** interface (public API). Implemented by `*InferFlow`. |
| **init.go** | Singleton init: `InitV1ConfigHandler()` returns Config. |
| **models.go** | Request/response types, payloads, and shared structs. |
| **inferflow.go** | **InferFlow** struct, `InitV1ConfigHandler()`, and **public entrypoints** that implement Config (e.g. `Onboard`, `Review`, `Promote`, `Edit`, `Clone`, `Delete`, `ScaleUp`, `Cancel`, `GetAll`, `GetAllRequests`, `ValidateRequest`, `GenerateFunctionalTestRequest`, `ExecuteFuncitonalTestRequest`, `GetLatestRequest`, `GetLoggingTTL`, `GetFeatureSchema`). |
| **inferflow_constants.go** | All `const` values: request types, statuses, method names, defaults, delimiters, etc. |
| **inferflow_review.go** | Review/approval and DB/ETCD write flow: `handleRejectedRequest`, `handleApprovedRequest`, rollback helpers (`rollbackApprovedRequest`, `rollbackPromoteRequest`, `rollbackEditRequest`, `rollbackCreatedConfigs`, `rollbackDeletedConfigs`), and create/update helpers (`createOrUpdateDiscoveryConfig`, `createOrUpdateInferFlowConfig`, `createOrUpdateEtcdConfig`). |
| **inferflow_fetch.go** | Batch fetch helpers used by `GetAll`: `batchFetchDiscoveryConfigs`, `batchFetchRingMasterConfigs`. |
| **inferflow_validation.go** | Package-level `ValidateInferFlowConfig`; method `ValidateOnboardRequest` (used by Onboard/Edit/Clone). |
| **inferflow_functional_testing.go** | Functional test entrypoints: `GenerateFunctionalTestRequest`, `ExecuteFuncitonalTestRequest` (gRPC/proto and test-result update logic). |
| **inferflow_helpers.go** | Small helpers: `GetDerivedConfigID`, `GetLoggingTTL`. |
| **adaptor.go** | DB/ETCD payload adaptors: `AdaptToEtcdInferFlowConfig`, `AdaptOnboardRequestToDBPayload`, `AdaptFromDbToInferFlowConfig`, `AdaptFromDbToConfigMapping`, etc. |
| **config_builder.go** | Build inferflow config from request: `GetInferflowConfig`, component/ranker/reranker building, and related constants (e.g. `COLON_DELIMITER`, `PIPE_DELIMITER`, `MODEL_FEATURE`). |
| **schema_adapter.go** | Feature schema from inferflow config: `BuildFeatureSchemaFromInferflow`, `ProcessResponseConfigFromInferflow`. |
| **proto/** | Inferflow gRPC proto and generated Go. |
| **inferflow_test.go** | Unit tests for the handler (same package). |

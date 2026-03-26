# Predator

Predator handles model lifecycle operations: onboarding, approval workflows, validation, fetch/list, upload from local or GCS, and functional testing.

## Package layout

```
internal/predator/
├── controller/ (wires routes to handler)
├── handler/ (Config implementation + helpers)
├── route/ 
├── proto/ and generated Go
├── init.go
└── README.md 
```

---

## Handler package structure

| File | Purpose |
|------|--------|
| **config.go** | Defines the **Config** interface (public API). Implemented by `*Predator`. |
| **init.go** | Singleton init: `InitV1ConfigHandler()` returns Config. |
| **model.go** / **models.go** | Request/response types, payloads, and shared structs. |
| **model_config.pb.go** | Generated from `proto/` (e.g. Triton model config). |
| **predator.go** | **Predator** struct, `InitV1ConfigHandler()`, and **public entrypoints** that implement Config (e.g. `HandleModelRequest`, `HandleDeleteModel`, `ProcessRequest`, `FetchModelConfig`, `FetchModels`, `ValidateRequest`, `GenerateFunctionalTestRequest`, `ExecuteFunctionalTestRequest`, `SendLoadTestRequest`, `UploadModelFolderFromLocal`, etc.). Remaining helpers used only here (e.g. `convertFields`, `convertInputWithFeatures`, `createModelParamsResponse`) also live in this file. |
| **predator_constants.go** | All `const` values: error messages, request types, stage names, config keys, etc. |
| **predator_approval.go** | Approval workflow: `processRequest`, onboard/scale-up/promote/delete/edit flows, GCS clone stages, DB population, restart deployable, revert, `createDiscoveryAndPredatorConfigTx`, `createPredatorConfigTx`, and related helpers. |
| **predator_validation.go** | Delete validation and async validation job: `ValidateDeleteRequest`, `validateEnsembleChildGroupDeletion`, lock release, `performAsyncValidation`, health checking, `clearTemporaryDeployable`, `copyExistingModelsToTemporary`, `copyRequestModelsToTemporary`, `restartTemporaryDeployable`, and related helpers. |
| **predator_fetch.go** | Fetch/list support: `batchFetchRelatedData`, `batchFetchDeployableConfigs`, `buildModelResponses` (used by `FetchModels`). |
| **predator_upload.go** | Upload-from-local and GCS sync: `uploadSingleModel`, `copyConfigToProdConfigSource`, `copyConfigToNewNameInConfigSource`, validation (metadata, online/offline/pricing features), `syncModelFiles`, `syncFullModel`, `syncPartialFiles`, `validateModelConfiguration`, `cleanEnsembleScheduling`, `replaceModelNameInConfigPreservingFormat`, and related helpers. |
| **predator_functional_testing.go** | Functional and load-test helpers: `flattenInputTo3DByteSlice`, `getElementSize`, `reshapeDataForBatch`, `convertDimsToIntSlice` (used by `GenerateFunctionalTestRequest`, `ExecuteFunctionalTestRequest`, `SendLoadTestRequest`). |
| **predator_helpers.go** | Shared helpers: GCS path parsing (`parseGCSURL`, `extractGCSPath`, `extractGCSDetails`), `GetDerivedModelName`, `GetOriginalModelName`, `isNonProductionEnvironment`. |
| **predator_test.go** | Unit tests for the handler (same package; no build tags). |


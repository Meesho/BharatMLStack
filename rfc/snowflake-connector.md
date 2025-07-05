# RFC: Snowflake Connector for BharatMLStack

## Metadata
- **Author:** _<your-name>_
- **Date Created:** <!-- YYYY-MM-DD -->
- **Status:** Draft
- **Target Release:** v1.1.0
- **Discussion Channel:** #connectors-snowflake

---

## 1. Summary
This proposal introduces native support for **Snowflake** as an offline feature data source inside BharatMLStack. The connector leverages the official `snowflake.ml.feature_store` Python API to read feature tables stored in Snowflake and plugs seamlessly into the existing abstraction layer implemented in `online-feature-store/examples/notebook/data_helpers.py`. The goal is to give data scientists the same zero-code experience they already enjoy with Parquet, Delta, and Hive tables while tapping into Snowflake's elastic compute and governance features.

## 2. Motivation
1. **Industry Adoption** – Snowflake is the de-facto cloud data warehouse for many ML teams.
2. **Unified Governance** – Keeping features in Snowflake allows teams to reuse established data contracts, lineage, and RBAC.
3. **Performance & Scale** – Snowflake's automatic scaling matches BharatMLStack's design philosophy of elastic cost.
4. **Developer Ergonomics** – The `snowflake.ml.feature_store` SDK abstracts low-level SQL and improves developer productivity.

Without this connector, users have to export data to object storage before ingesting it, introducing latency and operational overhead.

## 3. Goals
- Add **`SNOWFLAKE_TABLE`** as a new `source_type` value consumed by `read_from_source()`.
- Use the **`snowflake.ml.feature_store.FeatureStore`** abstraction for reading data, not raw JDBC.
- Support predicate-pushdown on a configurable partition column (default: `ingestion_timestamp`).
- Preserve existing semantics: the helper must return a Spark DataFrame deduplicated on entity keys.
- Provide clear configuration & credential management guidelines.

## 4. Non-Goals
- Online serving from Snowflake (future work).
- Bi-directional sync or writing features back into Snowflake.
- Support for Snowflake external tables or stages (out of scope for first version).

## 5. Detailed Design
### 5.1 Data Access Flow
1. **Session Creation** – A `snowpark.Session` is established using env-vars or a passed-in config map.
2. **Feature Store Bind** – Instantiate `FeatureStore(session)`.
3. **Table Selection** – Retrieve the maximum value of the partition column via the Feature Store API.
4. **Data Retrieval** – Load the filtered rows into a Spark DataFrame through `feature_store.read().to_spark()`.
5. **Schema Alignment** – Column names are normalized and optionally renamed according to `feature_mapping`.

### 5.2 API Touch-Points
| Function | Change | Rationale |
| --- | --- | --- |
| `read_from_source()` | Add `elif source_type == "SNOWFLAKE_TABLE"` branch to perform steps in 5.1 | Aligns with existing connector pattern |
| `src_type_to_partition_col_map` | Append `"SNOWFLAKE_TABLE": "ingestion_timestamp"` | Ensures incremental loads |

### 5.3 Configuration
| Parameter | Description | Example |
| --- | --- | --- |
| `SNOWFLAKE_ACCOUNT` | Snowflake account identifier | `xy12345.us-east-1` |
| `SNOWFLAKE_USER` | Login user | `ml_readonly` |
| `SNOWFLAKE_PASSWORD` | Password or OAuth token | `****` |
| `SNOWFLAKE_WAREHOUSE` | Virtual warehouse used for queries | `ML_WH` |
| `SNOWFLAKE_DATABASE` | Database containing feature tables | `FEATURE_DB` |
| `SNOWFLAKE_SCHEMA` | Default schema | `PUBLIC` |

Credentials are loaded by helper utilities and injected into the Snowpark session. Users may override through function arguments for multi-account scenarios.

### 5.4 Error Handling & Observability
- **Retry Logic:** Exponential backoff for transient Snowflake errors (`XYZ-0001`, `XYZ-0002`).
- **Metrics:**
  - `snowflake.connector.latency_ms`
  - `snowflake.connector.rows_read`
  - `snowflake.connector.bytes_scanned`
- **Logging:** Obfuscate sensitive parameters before logging connection strings.

## 6. Test Plan
1. **Unit Tests** – Mock `snowflake.ml.feature_store` to simulate data fetches.
2. **Integration Tests** – Use Snowflake's `TPCH_SF1` sample to validate partition filtering, column renaming, and null-handling.
3. **Schema Drift Tests** – Verify connector surface errors when expected columns are missing or types change.

## 7. Rollout Strategy
| Phase | Description |
| --- | --- |
| Alpha | Flag-guarded release for internal users. Collect feedback. |
| Beta | Publicly documented. Backwards compatibility guaranteed. |
| GA | Remove feature flag. Add SLA monitoring. |

## 8. Alternatives Considered
- **Spark JDBC Connector** – Simpler, but lacks feature-store metadata awareness and type mapping.
- **Materializing Views to Parquet** – Adds latency and duplication of storage.

## 9. Risks & Mitigations
| Risk | Impact | Mitigation |
| --- | --- | --- |
| Snowflake API instability | Ingest failures | Pin to major version; add compatibility tests |
| Credential leakage | Security breach | Use secrets manager integration; no plaintext logs |
| Large data pulls | Cost overruns | Enforce partition filters; sample row counts during CI |

## 10. Success Metrics
- < 2x latency compared to internal Parquet connector for same dataset size.
- 100% schema parity with Snowflake source tables.
- Adoption by at least two production teams within one quarter.

## 11. Future Work
- Online Snowflake serving connector.
- Support for Snowflake external tables & stages.
- Automatic statistics push to BharatML observability dashboard.

## 12. References
- Snowflake ML Feature Store Documentation (2024-04) — *"snowflake.ml.feature_store"*.
- BharatMLStack Architecture Overview (internal confluence).

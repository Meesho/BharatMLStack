# Feature Computation Platform — User Guide

This guide walks you through setting up and using the BharatML Stack Feature Computation Platform from scratch. It covers infrastructure setup, the data scientist workflow, and the admin review workflow.

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Prerequisites](#2-prerequisites)
3. [Setting Up Horizon (Go Control Plane)](#3-setting-up-horizon)
4. [Setting Up the Python SDK](#4-setting-up-the-python-sdk)
5. [Setting Up Trufflebox (Admin UI)](#5-setting-up-trufflebox)
6. [Setting Up GitHub Integration](#6-setting-up-github-integration)
7. [Setting Up CI](#7-setting-up-ci)
8. [Setting Up Airflow](#8-setting-up-airflow)
9. [Data Scientist Workflow](#9-data-scientist-workflow)
    - [Writing Features](#91-writing-features)
    - [Local Development & Testing](#92-local-development--testing)
    - [Pushing a PR](#93-pushing-a-pr)
    - [What Happens After Merge](#94-what-happens-after-merge)
10. [Admin Workflow](#10-admin-workflow)
    - [Reviewing a Feature PR](#101-reviewing-a-feature-pr)
    - [Operational Controls](#102-operational-controls)
11. [CLI Reference](#11-cli-reference)
12. [Concepts Reference](#12-concepts-reference)

---

## 1. Architecture Overview

The platform has three systems:

```
┌────────────────────────────────────────────────────────────────────────┐
│                                                                        │
│   Data Scientist                   CI (GitHub Actions)                 │
│   ┌────────────┐                  ┌─────────────────┐                  │
│   │  @asset()  │───── PR ────────▶│ bharatml manifest│                  │
│   │  notebooks │                  │ bharatml diff    │                  │
│   └────────────┘                  └────────┬────────┘                  │
│                                            │ manifest.json             │
│                                            ▼                           │
│                              ┌────────────────────────┐                │
│                              │  Trufflebox (Admin UI)  │                │
│                              │  Feature Review Panel   │                │
│                              └────────────┬───────────┘                │
│                                           │                            │
│                    ┌──────────────────────┬┴─────────────┐             │
│                    ▼                      ▼               ▼             │
│             ┌────────────┐        ┌────────────┐  ┌────────────┐       │
│             │  Horizon   │        │ GitHub API │  │  Airflow   │       │
│             │  (Go API)  │        │ (approve/  │  │  (DAGs)    │       │
│             └─────┬──────┘        │  merge)    │  └─────┬──────┘       │
│                   │               └────────────┘        │              │
│                   ▼                                     ▼              │
│             ┌──────────────────────────────────────────────┐           │
│             │           Databricks / Spark                  │           │
│             │  NotebookRuntime.run() → compute features     │           │
│             └──────────────────────────────────────────────┘           │
└────────────────────────────────────────────────────────────────────────┘
```

| System | Role |
|--------|------|
| **Horizon** (Go) | Control plane — asset registry, DAG, necessity, caching, execution plans |
| **Python SDK** | Authoring (`@asset`), runtime (`NotebookRuntime`), CLI (`bharatml`) |
| **Trufflebox** | Admin UI for reviewing and approving feature changes |
| **Airflow** | Scheduling and orchestrating Databricks notebook runs |

---

## 2. Prerequisites

| Component | Requirement |
|-----------|-------------|
| Go | >= 1.21 |
| Python | >= 3.9 |
| Node.js | >= 16 |
| PostgreSQL | Any recent version (for Horizon's database) |
| Databricks | Workspace with notebook support and Spark |
| Airflow | >= 2.x with `apache-airflow-providers-databricks` |
| GitHub | Repository for feature notebooks, with webhook support |

---

## 3. Setting Up Horizon

Horizon is the Go backend that manages the asset registry, dependency DAG, necessity resolution, and execution planning.

### 3.1 Database Setup

Create a PostgreSQL database for Horizon. The tables are auto-created via `CREATE TABLE IF NOT EXISTS` statements when Horizon starts.

```sql
CREATE DATABASE horizon;
CREATE USER horizon WITH PASSWORD '<your-password>';
GRANT ALL PRIVILEGES ON DATABASE horizon TO horizon;
```

Six tables will be created automatically:

| Table | Purpose |
|-------|---------|
| `asset_specs` | Asset definitions (name, entity, schedule, trigger, serving) |
| `asset_dependencies` | DAG edges between assets |
| `asset_necessity` | Computed necessity state per asset |
| `dataset_partitions` | External dataset readiness tracking |
| `materializations` | Content-addressed computation cache |
| `feature_reviews` | PR review workflow state |

### 3.2 Configuration

Set the following environment variables:

```bash
# Database
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=horizon
export DB_USER=horizon
export DB_PASSWORD=<password>

# Feature Compute Engine paths
export FCE_ARTIFACT_BASE_PATH=/data/artifacts       # where computed artifacts are stored
export FCE_SERVING_BASE_PATH=/data/serving           # where serving tables are published
export FCE_DEFAULT_PARTITION=ds                      # default partition key

# Feature Review / GitHub integration
export FCE_FEATURE_REPOS=your-org/feature-repo       # comma-separated list of repos to watch
export FCE_WEBHOOK_SECRET=<github-webhook-secret>     # shared secret for webhook verification
export FCE_HORIZON_INTERNAL_URL=http://localhost:8080  # Horizon's own URL (for internal calls)
export GITHUB_TOKEN=<github-pat>                       # GitHub token for API calls
```

### 3.3 Build and Run

```bash
cd horizon/
go mod tidy
go build -o horizon ./cmd/horizon/
./horizon
```

Horizon starts on port `8080` by default. Verify it's running:

```bash
curl http://localhost:8080/api/v1/fce/necessity
# Should return: {"necessities": {}}
```

---

## 4. Setting Up the Python SDK

The SDK provides the `@asset` decorator, `NotebookRuntime`, and the `bharatml` CLI.

### 4.1 Install

```bash
cd py-sdk/feature_compute_engine/
pip install -e .
```

### 4.2 Verify

```bash
bharatml --help
```

Expected output:

```
usage: bharatml [-h] [--verbose]
                {manifest,diff,apply,plan,lineage,serving,necessity,backfill} ...

Feature Computation Platform CLI
```

### 4.3 Install in Databricks

Upload the `feature_compute_engine` package to your Databricks workspace or install it as a cluster library:

```bash
pip install /path/to/feature_compute_engine/
```

---

## 5. Setting Up Trufflebox

Trufflebox is the admin UI for reviewing feature PRs.

### 5.1 Configuration

```bash
cd trufflebox-ui/

# Point to Horizon
export REACT_APP_HORIZON_BASE_URL=http://localhost:8080

# Enable the Feature Review section
export REACT_APP_FEATURE_REVIEW_ENABLED=true
```

### 5.2 Run

```bash
npm install
npm start
```

The UI is available at `http://localhost:3000`. The **Feature Reviews** section appears in the sidebar for users with the `admin` role.

---

## 6. Setting Up GitHub Integration

### 6.1 Create a GitHub Webhook

On each repository listed in `FCE_FEATURE_REPOS`, configure a webhook:

| Setting | Value |
|---------|-------|
| **Payload URL** | `https://<horizon-host>/webhooks/github` |
| **Content type** | `application/json` |
| **Secret** | Same value as `FCE_WEBHOOK_SECRET` |
| **Events** | Pull requests |

### 6.2 GitHub Token

The token set in `GITHUB_TOKEN` needs these permissions on the watched repos:
- `contents: read` (fetch manifest files)
- `pull_requests: write` (approve and merge PRs)
- `issues: write` (post PR comments)

---

## 7. Setting Up CI

Copy the CI workflow to each feature notebook repository:

```yaml
# .github/workflows/feature-ci.yml
# Triggers on PRs that touch notebooks/ or the SDK
```

The workflow (already provided at `.github/workflows/feature-ci.yml`) does:

1. Installs the SDK
2. Runs `bharatml manifest` to extract asset metadata from `@asset` decorators
3. Runs `bharatml diff` to compare against the base branch
4. Commits `manifest.json` to the PR branch
5. Posts a summary comment on the PR

**Key rule: CI never calls Horizon.** The manifest is a pure JSON file generated locally.

---

## 8. Setting Up Airflow

### 8.1 Install the SDK in Airflow

```bash
pip install /path/to/feature_compute_engine/
pip install apache-airflow-providers-databricks
```

### 8.2 Configure Airflow Connections

Create two Airflow connections:

**Horizon connection** (`horizon_default`):
| Field | Value |
|-------|-------|
| Conn Type | HTTP |
| Host | `horizon-host` |
| Port | `8080` |
| Password | API key (if configured) |

**Databricks connection** (`databricks_default`):
| Field | Value |
|-------|-------|
| Conn Type | Databricks |
| Host | `<workspace-url>` |
| Password | Databricks PAT |

### 8.3 Generate DAGs

After assets are registered in Horizon (via the merge flow), generate DAG files:

```bash
python -m feature_compute_engine.airflow.dag_generator \
    --horizon-url http://horizon:8080 \
    --output-dir /opt/airflow/dags/fce/
```

This creates one DAG file per `(notebook, trigger_type)` combination:
- `fce__user_features__schedule_3h.py`
- `fce__user_features__upstream.py`
- `fce__product_features__schedule_24h.py`

Each DAG contains a single `AssetExecutionOperator` task. Airflow picks up the files automatically.

### 8.4 Create the Databricks Pool

```bash
airflow pools set databricks_pool 10 "Databricks concurrent jobs"
```

---

## 9. Data Scientist Workflow

### 9.1 Writing Features

Create one Python file per entity in the `notebooks/` directory. Each file can contain multiple `@asset` decorated functions.

**Example: `notebooks/user_features.py`**

```python
from feature_compute_engine import asset, Input

@asset(
    name="fg.user_spend",
    entity="user",
    entity_key="user_id",
    schedule="3h",
    serving=True,
    inputs=[Input("silver.orders", partition="ds")],
    checks=["row_count > 0", "null_rate(user_id) < 0.001"],
)
def user_spend(ctx, ds, orders_df):
    """Total spend per user, refreshed every 3 hours."""
    from pyspark.sql import functions as F

    return orders_df.groupBy("user_id").agg(
        F.sum("amount").alias("total_spend"),
        F.count("order_id").alias("order_count"),
        F.max("order_date").alias("last_order_date"),
    )


@asset(
    name="fg.user_spend_7d",
    entity="user",
    entity_key="user_id",
    trigger="upstream",
    serving=True,
    inputs=[Input("fg.user_spend", partition="ds", window=7)],
    checks=["row_count > 0"],
)
def user_spend_7d(ctx, ds, spend_7d_df):
    """Rolling 7-day spend aggregation, triggered when user_spend completes."""
    from pyspark.sql import functions as F

    return spend_7d_df.groupBy("user_id").agg(
        F.sum("total_spend").alias("total_spend_7d"),
        F.avg("total_spend").alias("avg_daily_spend_7d"),
        F.sum("order_count").alias("order_count_7d"),
    )
```

At the bottom of the notebook, add the runtime entry point:

```python
if __name__ == "__main__":
    from feature_compute_engine import NotebookRuntime
    NotebookRuntime.run()
```

#### `@asset` Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | str | required | Unique asset name (convention: `fg.<feature_group>`) |
| `entity` | str | required | Entity type: `user`, `product`, `order`, etc. |
| `entity_key` | str | required | Column name that identifies the entity (e.g., `user_id`) |
| `partition` | str | `"ds"` | Partition key (usually date string) |
| `schedule` | str | None | Cron-like schedule: `"1h"`, `"3h"`, `"6h"`, `"12h"`, `"24h"` |
| `trigger` | str | `"upstream"` | `"upstream"` (runs when inputs are ready) or auto-set from `schedule` |
| `serving` | bool | `True` | If true, output is published to serving tables for online access |
| `incremental` | bool | `False` | If true, only processes new data since last run |
| `freshness` | str | None | Expected freshness SLA (e.g., `"6h"`) |
| `inputs` | list | `[]` | List of `Input(...)` dependencies |
| `checks` | list | `[]` | Post-compute data quality checks |

#### `Input` Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `name` | str | required | Source asset or dataset name |
| `partition` | str | `"ds"` | Partition key |
| `window` | int | None | Number of past partitions to include (e.g., `7` for 7-day window) |
| `input_type` | str | `"internal"` | `"internal"` (other assets) or `"external"` (raw sources) |

#### Function Signature

```python
def compute_fn(ctx: AssetContext, partition_value: str, *input_dataframes) -> DataFrame:
```

- `ctx` — runtime context with partition info, compute key, artifact path
- `partition_value` — the partition being computed (e.g., `"2026-03-01"`)
- `*input_dataframes` — one DataFrame per declared input, **in declaration order**

#### Data Quality Checks

Checks are string expressions evaluated against the output DataFrame:

| Expression | Example | Description |
|------------|---------|-------------|
| `row_count` | `"row_count > 0"` | Total row count |
| `null_rate(col)` | `"null_rate(user_id) < 0.001"` | Fraction of nulls in a column |
| `distinct_count(col)` | `"distinct_count(user_id) > 100"` | Number of distinct values |

If any check fails, the asset is marked as failed and is not published.

### 9.2 Local Development & Testing

Use `DevContext` for interactive development in a Databricks notebook or local Spark session:

```python
from feature_compute_engine import DevContext
from pyspark.sql import SparkSession

spark = SparkSession.builder.getOrCreate()

# Load test data
orders_df = spark.read.table("silver.orders").filter("ds = '2026-03-01'")

# Create dev context
ctx = DevContext(partition="2026-03-01")

# Call the asset function directly
result = user_spend(ctx, "2026-03-01", orders_df)
result.show()
result.printSchema()
```

`DevContext` has sensible defaults (`asset_name="dev"`, `necessity="active"`) so your function runs without needing a Horizon connection.

### 9.3 Pushing a PR

```bash
git checkout -b feat/add-user-spend
git add notebooks/user_features.py
git commit -m "Add user spend feature group"
git push origin feat/add-user-spend
```

Open a PR. The following happens automatically:

1. **CI runs** — GitHub Actions imports your notebook, runs `bharatml manifest`, and generates `.bharatml/manifest.json`
2. **CI posts a PR comment** with a summary:
   ```
   ## bharatml manifest
   Assets: 2 total (2 serving)
   Entities: user(2)
   Triggers: schedule_3h(1), upstream(1)

   ### Changes vs main
   NEW: fg.user_spend, fg.user_spend_7d
   ```
3. **Manifest is committed** to the PR branch automatically
4. **GitHub webhook** notifies Horizon, which creates a pending review in Trufflebox

At this point, you wait for an admin to review and approve in Trufflebox.

### 9.4 What Happens After Merge

Once an admin approves and merges (see [Admin Workflow](#10-admin-workflow)):

1. Horizon registers your assets and resolves the dependency DAG
2. Necessity is computed — your assets become `ACTIVE` (since `serving=True`)
3. Airflow DAGs are generated (e.g., `fce__user_features__schedule_3h.py`)
4. Every 3 hours, Airflow triggers the DAG:
   - Asks Horizon for an execution plan
   - Launches your notebook on Databricks
   - `NotebookRuntime.run()` executes only the assets that need work
   - Results are reported back to Horizon
   - `fg.user_spend_7d` (trigger=upstream) fires automatically after `fg.user_spend` completes

You don't need to do anything after merge — the platform handles scheduling, caching, and trigger cascading.

---

## 10. Admin Workflow

### 10.1 Reviewing a Feature PR

#### Step 1: Open Feature Reviews

Navigate to **Feature Reviews** in the Trufflebox sidebar (admin-only). You see a list of pending reviews with:
- PR number, title, author, repo
- Status badge
- Asset summary (new / modified / removed)

#### Step 2: Compute the Plan

Click **"Show Plan"** on a pending review (or navigate to the detail page and click it there).

Trufflebox fetches the manifest from the PR branch and asks Horizon to compute the impact analysis. The review status moves from `pending` → `plan_computed`.

The plan shows:

| Section | What It Shows |
|---------|---------------|
| **Changes** | Which assets are added, modified, or removed |
| **Necessity Changes** | How necessity states change (e.g., new ACTIVE assets) |
| **Recomputation Impact** | Which partitions need recomputing, estimated cost in USD |
| **Airflow DAGs** | Which DAGs will be created or updated |
| **Warnings** | Circular dependencies, missing inputs, or other issues |

#### Step 3: Approve or Reject

- **Approve**: Optionally add a comment, then click **"Approve"**. This approves the PR on GitHub.
- **Reject**: Provide a required comment explaining why, then click **"Reject"**. A comment is posted on the PR.
- **Recompute Plan**: If the PR is updated, click to recompute with the latest changes.

#### Step 4: Merge

After approval, click **"Merge"**. This merges the PR on GitHub. On merge:

1. Horizon registers the assets from the manifest
2. The dependency DAG is rebuilt
3. Necessity is resolved for all assets
4. Airflow DAGs are generated

The review status moves to `merged`.

#### Step 5: Review History

Visit **Feature Reviews > History** for a searchable, sortable table of all past reviews.

### 10.2 Operational Controls

#### View Necessity States

```bash
bharatml necessity --horizon-url http://horizon:8080
```

Output:
```
fg.user_spend                            active
fg.user_spend_7d                         active
fg.user_temp                             skipped
```

#### Override Serving

Force an asset's serving flag on or off:

```bash
bharatml serving set fg.user_temp \
    --serving true \
    --reason "Needed for A/B test" \
    --by admin@example.com \
    --horizon-url http://horizon:8080
```

This overrides the declared `serving` value. The necessity resolver re-runs automatically.

#### View Lineage

```bash
bharatml lineage --horizon-url http://horizon:8080
bharatml lineage fg.user_spend --upstream --horizon-url http://horizon:8080
bharatml lineage fg.user_spend --downstream --horizon-url http://horizon:8080
```

#### Compute Impact Plan (CLI)

Preview the impact of a manifest without applying it:

```bash
bharatml plan \
    --manifest .bharatml/manifest.json \
    --horizon-url http://horizon:8080 \
    --format text
```

#### Apply Manifest (CLI)

Register assets and generate DAGs directly (bypasses the UI review workflow):

```bash
bharatml apply \
    --manifest .bharatml/manifest.json \
    --horizon-url http://horizon:8080 \
    --dag-output-dir /opt/airflow/dags/fce/
```

#### Mark External Dataset Ready

When an external dataset partition is available:

```bash
curl -X POST http://horizon:8080/api/v1/fce/datasets/ready \
    -H "Content-Type: application/json" \
    -d '{"dataset_name": "silver.orders", "partition": "2026-03-01"}'
```

This evaluates downstream triggers and may fire asset computation.

#### Run Backfill

Recompute an asset across a date range:

```bash
bharatml backfill fg.user_spend \
    --range 2026-01-01..2026-02-01 \
    --horizon-url http://horizon:8080
```

---

## 11. CLI Reference

| Command | Network Required | Description |
|---------|-----------------|-------------|
| `bharatml manifest` | No | Extract `@asset` specs → `.bharatml/manifest.json` |
| `bharatml diff` | No | Compare two manifests locally |
| `bharatml plan` | Yes (Horizon) | Compute impact analysis |
| `bharatml apply` | Yes (Horizon) | Register assets + generate Airflow DAGs |
| `bharatml lineage` | Yes (Horizon) | View dependency graph |
| `bharatml serving set` | Yes (Horizon) | Override serving flag |
| `bharatml necessity` | Yes (Horizon) | View all necessity states |
| `bharatml backfill` | Yes (Horizon) | Recompute across a date range |

---

## 12. Concepts Reference

### Necessity States

Every registered asset is assigned a necessity state:

| State | Meaning | What Happens |
|-------|---------|-------------|
| **ACTIVE** | `serving=true` or has a serving override | Computed and published to serving tables |
| **TRANSIENT** | `serving=false` but a downstream ACTIVE asset depends on it | Computed and stored as artifact only (not served) |
| **SKIPPED** | Not needed by any ACTIVE or TRANSIENT asset | Not computed at all |

### Compute Key (Content Addressing)

```
compute_key = hash(asset_version + sorted(input_versions) + params + env_fingerprint)
```

If a materialization with the same compute key already exists in the artifact store, execution is skipped (cache hit). This means:
- Identical code + identical inputs = no recomputation
- Changing code or any input version = new compute key = recomputation

### Trigger Types

| Type | When It Fires |
|------|--------------|
| `schedule_1h` | Every hour |
| `schedule_3h` | Every 3 hours |
| `schedule_6h` | Every 6 hours |
| `schedule_12h` | Every 12 hours |
| `schedule_24h` | Daily at 2 AM |
| `upstream` | When all declared inputs are ready for the partition |

### Storage Layout

```
Artifact Store:
  /_artifacts/{asset_name}/{partition}/{compute_key}/

Serving Tables:
  /serving/{asset_name}/{partition}/
```

- **Artifacts** are kept for all computed outputs (history, rollback)
- **Serving tables** only contain the latest output for ACTIVE assets
- **Online store** (existing system) syncs from serving tables

### Execution Plan Actions

When Airflow asks Horizon "what should I run?", each asset gets one of:

| Action | Meaning |
|--------|---------|
| `execute` | Cache miss + necessary → run the function |
| `skip_cached` | Compute key matches existing materialization → skip |
| `skip_unnecessary` | Necessity is SKIPPED → don't compute |

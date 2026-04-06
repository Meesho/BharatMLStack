"""
OSS replacement for Airflow/Databricks: calls skye-admin for job metadata,
generates random embedding data, and produces to the Kafka topic.
No GCS or Spark; used for quick-start / demo.
"""
import argparse
import json
import logging
import os
import random
import sys
from typing import Any

import requests
from confluent_kafka import Producer

# Logging: level from LOG_LEVEL env (DEBUG, INFO, WARNING, ERROR), default INFO
LOG_LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    format="%(asctime)s %(levelname)s [skye-trigger] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    stream=sys.stdout,
)
logger = logging.getLogger(__name__)

# Defaults for Docker quick-start
DEFAULT_ADMIN_URL = os.environ.get("SKYE_ADMIN_URL", "http://skye-admin:8092")
DEFAULT_KAFKA_BOOTSTRAP = os.environ.get("KAFKA_BOOTSTRAP_SERVERS", "broker:29092")
DEFAULT_ENVIRONMENT = os.environ.get("SKYE_ENVIRONMENT", "local")
DEFAULT_VECTOR_DB_TYPE = "QDRANT"
DEFAULT_NUM_POINTS = int(os.environ.get("SKYE_TRIGGER_NUM_POINTS", "10000"))
DEFAULT_EMBEDDING_DIM = int(os.environ.get("SKYE_EMBEDDING_DIM", "64"))


def call_admin_process_multi_variant(
    admin_url: str,
    entity: str,
    model: str,
    variants: list[str],
    vector_db_type: str,
) -> dict[str, Any]:
    """Call skye-admin POST /api/v1/qdrant/process-multi-variant; return response JSON."""
    url = f"{admin_url.rstrip('/')}/api/v1/qdrant/process-multi-variant"
    payload = {
        "entity": entity,
        "model": model,
        "variants": variants,
        "vector_db_type": vector_db_type,
    }
    logger.info(
        "Calling skye-admin url=%s entity=%s model=%s variants=%s",
        url, entity, model, variants,
    )
    try:
        resp = requests.post(url, json=payload, timeout=30)
        if not resp.ok:
            try:
                err_body = resp.text
            except Exception:
                err_body = ""
            logger.error(
                "skye-admin request failed url=%s entity=%s model=%s status=%s response_body=%s",
                url, entity, model, resp.status_code, err_body[:500] if err_body else "",
            )
        resp.raise_for_status()
        data = resp.json()
        logger.info(
            "skye-admin response status=%d topic_name=%s embedding_store_version=%s number_of_partitions=%s",
            resp.status_code,
            data.get("topic_name"),
            data.get("embedding_store_version"),
            data.get("number_of_partitions"),
        )
        return data
    except requests.RequestException as e:
        if getattr(e, "response", None) is None:
            logger.error(
                "skye-admin request failed url=%s entity=%s model=%s error=%s",
                url, entity, model, e,
                exc_info=True,
            )
        raise


def build_event(
    candidate_id: str,
    entity: str,
    model: str,
    environment: str,
    embedding_store_version: int,
    embedding_dim: int,
    variants_version_map: dict[str, int],
    partition: str = "",
) -> dict[str, Any]:
    """Build one embedding event matching skye consumer Event struct."""
    index_emb = [round(random.uniform(-1.0, 1.0), 6) for _ in range(embedding_dim)]
    search_emb = [round(random.uniform(-1.0, 1.0), 6) for _ in range(embedding_dim)]
    variants_index_map = {v: True for v in variants_version_map}
    return {
        "candidate_id": candidate_id,
        "entity": entity,
        "model_name": model,
        "environment": environment,
        "embedding_store_version": embedding_store_version,
        "partition": partition,
        "index_space": {
            "embedding": index_emb,
            "variants_version_map": variants_version_map,
            "variants_index_map": variants_index_map,
            "operation": "ADD",
            "payload": {"portfolio_id": "0"},
        },
        "search_space": {"embedding": search_emb},
    }


def run_job(
    entity: str,
    model: str,
    variants: list[str],
    environment: str = DEFAULT_ENVIRONMENT,
    vector_db_type: str = DEFAULT_VECTOR_DB_TYPE,
    admin_url: str = DEFAULT_ADMIN_URL,
    kafka_bootstrap: str = DEFAULT_KAFKA_BOOTSTRAP,
    num_points: int = DEFAULT_NUM_POINTS,
    embedding_dim: int = DEFAULT_EMBEDDING_DIM,
) -> tuple[bool, str]:
    """
    Call skye-admin, generate random embedding events, produce to Kafka, send EOF.
    Returns (success, message).
    """
    logger.info(
        "run_job start entity=%s model=%s variants=%s environment=%s num_points=%s",
        entity, model, variants, environment, num_points,
    )
    if not variants:
        logger.error("run_job failed: variants list is required")
        return False, "variants list is required"

    # 1) Get job metadata from skye-admin
    try:
        resp = call_admin_process_multi_variant(
            admin_url, entity, model, variants, vector_db_type
        )
    except requests.RequestException as e:
        logger.error("run_job failed: skye-admin request failed error=%s", e)
        return False, f"skye-admin request failed: {e}"

    topic_name = resp.get("topic_name")
    embedding_store_version = int(resp.get("embedding_store_version", 0))
    variants_map = resp.get("variants") or {}
    if isinstance(variants_map, dict):
        variants_version_map = {k: int(v) for k, v in variants_map.items()}
    else:
        variants_version_map = {v: 1 for v in variants}
    num_partitions = max(1, int(resp.get("number_of_partitions", 1)))

    if not topic_name:
        logger.error("run_job failed: admin response missing topic_name resp_keys=%s", list(resp.keys()))
        return False, "admin response missing topic_name"

    logger.info(
        "Producing to Kafka topic=%s bootstrap=%s partitions=%s embedding_store_version=%s",
        topic_name, kafka_bootstrap, num_partitions, embedding_store_version,
    )

    # 2) Kafka producer (plaintext for quick-start). Do not pass partition= to produce();
    #    the topic may have fewer partitions than admin's number_of_partitions, which causes
    #    _UNKNOWN_PARTITION. Let the producer use key-based partitioning only.
    conf = {
        "bootstrap.servers": kafka_bootstrap,
        "client.id": "skye-trigger-oss",
    }
    try:
        producer = Producer(conf)
    except Exception as e:
        logger.error("run_job failed: Kafka producer init error=%s", e, exc_info=True)
        return False, f"Kafka producer init failed: {e}"

    # 3) Produce random embedding events (key only; no explicit partition)
    for i in range(num_points):
        evt = build_event(
            candidate_id=str(i),
            entity=entity,
            model=model,
            environment=environment,
            embedding_store_version=embedding_store_version,
            embedding_dim=embedding_dim,
            variants_version_map=variants_version_map,
        )
        value = json.dumps(evt).encode("utf-8")
        key = str(i % num_partitions).encode("utf-8")
        producer.produce(
            topic=topic_name,
            key=key,
            value=value,
        )
        if (i + 1) % 2000 == 0:
            producer.flush()
            logger.debug("Produced %s / %s events", i + 1, num_points)

    producer.flush()
    logger.info("Produced %s embedding events to topic=%s", num_points, topic_name)

    # 4) Send EOF per partition (consumer expects EOF to finish batch). Use key only.
    eof_evt = build_event(
        candidate_id="EOF",
        entity=entity,
        model=model,
        environment=environment,
        embedding_store_version=embedding_store_version,
        embedding_dim=embedding_dim,
        variants_version_map=variants_version_map,
    )
    eof_bytes = json.dumps(eof_evt).encode("utf-8")
    for p in range(num_partitions):
        producer.produce(
            topic=topic_name,
            key=str(p).encode("utf-8"),
            value=eof_bytes,
        )
    producer.flush()
    logger.info("Sent EOF (key 0..%s) to topic=%s", num_partitions - 1, topic_name)

    msg = f"Produced {num_points} events + EOF to topic {topic_name}"
    logger.info("run_job success %s", msg)
    return True, msg


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Trigger skye embedding job: call admin, produce random data to Kafka."
    )
    parser.add_argument("--entity", required=True, help="Entity name")
    parser.add_argument("--model", required=True, help="Model name")
    parser.add_argument(
        "--variants",
        required=True,
        help="Comma-separated variant names (e.g. organic,ads)",
    )
    parser.add_argument("--environment", default=DEFAULT_ENVIRONMENT)
    parser.add_argument("--vector-db-type", default=DEFAULT_VECTOR_DB_TYPE)
    parser.add_argument("--admin-url", default=DEFAULT_ADMIN_URL)
    parser.add_argument("--kafka-bootstrap", default=DEFAULT_KAFKA_BOOTSTRAP)
    parser.add_argument("--num-points", type=int, default=DEFAULT_NUM_POINTS)
    parser.add_argument("--embedding-dim", type=int, default=DEFAULT_EMBEDDING_DIM)
    args = parser.parse_args()

    variants_list = [v.strip() for v in args.variants.split(",") if v.strip()]
    ok, msg = run_job(
        entity=args.entity,
        model=args.model,
        variants=variants_list,
        environment=args.environment,
        vector_db_type=args.vector_db_type,
        admin_url=args.admin_url,
        kafka_bootstrap=args.kafka_bootstrap,
        num_points=args.num_points,
        embedding_dim=args.embedding_dim,
    )
    if ok:
        print(msg)
    else:
        print(msg, file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()

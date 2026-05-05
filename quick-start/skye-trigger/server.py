"""
Minimal HTTP server to trigger skye embedding jobs (OSS replacement for Airflow).
POST /trigger with JSON body: entity, model_name, variants (list or comma-sep string), optional env/vector_db_type.
"""
import logging
import os

from flask import Flask, jsonify, request

from run_embedding_job import (
    DEFAULT_ADMIN_URL,
    DEFAULT_KAFKA_BOOTSTRAP,
    DEFAULT_NUM_POINTS,
    run_job,
)

# Reuse same log format as run_embedding_job when run as server
LOG_LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    format="%(asctime)s %(levelname)s [skye-trigger] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    level=getattr(logging, LOG_LEVEL, logging.INFO),
)
logger = logging.getLogger(__name__)

app = Flask(__name__)


@app.route("/health", methods=["GET"])
def health():
    logger.debug("GET /health")
    return jsonify({"status": "ok"})


@app.route("/trigger", methods=["POST"])
def trigger():
    """
    Request body (JSON):
      entity (required), model_name (required), variants (required: list or comma-sep string)
      environment, vector_db_type, num_points optional.
    """
    body = request.get_json(silent=True) or {}
    if not body and request.data:
        logger.warning("POST /trigger received non-JSON or empty body")
    entity = body.get("entity") or body.get("entity_name")
    model = body.get("model") or body.get("model_name")
    raw_variants = body.get("variants") or body.get("variant_name") or ""
    if isinstance(raw_variants, list):
        variants = [str(v) for v in raw_variants]
    else:
        variants = [v.strip() for v in str(raw_variants).split(",") if v.strip()]

    logger.info(
        "POST /trigger entity=%s model=%s variants=%s",
        entity or "(missing)",
        model or "(missing)",
        variants,
    )

    if not entity or not model:
        logger.warning("POST /trigger validation failed: entity and model_name are required")
        return jsonify({"ok": False, "error": "entity and model_name are required"}), 400
    if not variants:
        logger.warning("POST /trigger validation failed: variants (list or comma-sep) is required")
        return jsonify({"ok": False, "error": "variants (list or comma-sep) is required"}), 400

    environment = body.get("environment", os.environ.get("SKYE_ENVIRONMENT", "local"))
    vector_db_type = body.get("vector_db_type", "QDRANT")
    admin_url = body.get("admin_url") or os.environ.get("SKYE_ADMIN_URL", DEFAULT_ADMIN_URL)
    kafka_bootstrap = body.get("kafka_bootstrap") or os.environ.get(
        "KAFKA_BOOTSTRAP_SERVERS", DEFAULT_KAFKA_BOOTSTRAP
    )
    num_points = int(body.get("num_points", os.environ.get("SKYE_TRIGGER_NUM_POINTS", DEFAULT_NUM_POINTS)))

    try:
        ok, msg = run_job(
            entity=entity,
            model=model,
            variants=variants,
            environment=environment,
            vector_db_type=vector_db_type,
            admin_url=admin_url,
            kafka_bootstrap=kafka_bootstrap,
            num_points=num_points,
        )
        if ok:
            logger.info("POST /trigger success entity=%s model=%s message=%s", entity, model, msg)
            return jsonify({"ok": True, "message": msg})
        logger.error("POST /trigger job failed entity=%s model=%s error=%s", entity, model, msg)
        return jsonify({"ok": False, "error": msg}), 502
    except Exception as e:
        logger.exception("POST /trigger unexpected error entity=%s model=%s error=%s", entity, model, e)
        return jsonify({"ok": False, "error": str(e)}), 500


if __name__ == "__main__":
    port = int(os.environ.get("PORT", "8080"))
    logger.info("Starting skye-trigger server port=%s log_level=%s", port, LOG_LEVEL)
    app.run(host="0.0.0.0", port=port)

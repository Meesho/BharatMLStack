#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "Creating virtual environment..."
python3 -m venv venv

echo "Activating virtual environment and installing dependencies..."
source venv/bin/activate
pip install --upgrade pip
pip install inference-logging-client==0.3.1

echo "Running asynclogparse.py..."
python3 asynclogparse.py /Users/neeharmavuduru/Downloads/Image_search_gcs-flush_pdp-ad-multitask-fieldaware-categorylevelscaleup_2026-04-03_16_pdp-ad-multitask-fieldaware-categorylevelscaleup--prd-model-proxy-service-bytes-primary-776f857d9c-sqxh8_2026-04-03_16-34-41.log

echo "Done!"


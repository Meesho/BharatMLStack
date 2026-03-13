#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "Creating virtual environment..."
python3 -m venv venv

echo "Activating virtual environment and installing dependencies..."
source venv/bin/activate
pip install --upgrade pip
pip install inference-logging-client

echo "Running asynclogparse.py..."
python3 asynclogparse.py /Users/neeharmavuduru/Downloads/Image_search_gcs-flush_pdp-ad-multitask-fieldaware-categorylevelscaleup_2026-03-12_15_pdp-ad-multitask-fieldaware-categorylevelscaleup_2026-03-12_15-04-12.log

echo "Done!"


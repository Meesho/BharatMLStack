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
python3 asynclogparse.py Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-13_17-26-43.log

echo "Done!"


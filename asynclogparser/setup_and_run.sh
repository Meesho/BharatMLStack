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
python3 asynclogparse.py /Users/neeharmavuduru/Downloads/Image_search_gcs-flush_search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-04-20_06_search-ad-head-multitask-fieldaware-categorylevelscaleup--logging-test_2026-04-20_06-20-15_0.log

echo "Done!"


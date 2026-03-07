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
python3 asynclogparse.py /Users/neeharmavuduru/Downloads/Image_search_gcs-flush_pdp-widget-ad-scaleup-100-rev-per-view-throttling-hourly-cal_2026-02-27_07-29-01.log

echo "Done!"


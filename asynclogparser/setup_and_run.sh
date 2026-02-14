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
python3 asynclogparse.py search-ad-head-multitask-fieldaware-categorylevelscaleup_2026-02-03_17-10-42.log

echo "Done!"


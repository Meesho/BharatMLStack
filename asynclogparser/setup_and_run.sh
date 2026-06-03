#!/bin/bash
set -e

cd "$(dirname "$0")"

if [ -z "$1" ]; then
    echo "Usage: ./setup_and_run.sh <path-to-log-file>"
    echo "Example: ./setup_and_run.sh ~/Downloads/my_service_2026-05-28.log"
    exit 1
fi
LOG_FILE="$1"
OUTPUT_CSV="${LOG_FILE%.log}.csv"

echo "Creating fresh virtual environment..."
rm -rf venv
python3 -m venv venv

echo "Activating virtual environment and installing dependencies..."
source venv/bin/activate
pip install --upgrade pip

# Install the in-repo SSD parser package (pulls inference-logging-client>=0.3.9 + pandas)
pip install -e ../inference-logging-client-ssd

echo ""
echo "Parsing: $LOG_FILE"
echo "Output:  $OUTPUT_CSV"
echo ""

python3 -c "
import sys
from pathlib import Path
from inference_logging_client_ssd import parse_log_file

log_path = sys.argv[1]
output_csv = sys.argv[2]

print('Parsing log file...')
df = parse_log_file(
    log_path,
    inference_host='http://horizon-v2.prd.meesho.int',
    api_path='/api/v1/horizon/inferflow-config-registry/get_feature_schema',
)

print()
print('=== DataFrame Summary ===')
print(f'Rows:    {len(df):,}')
print(f'Columns: {len(df.columns)}')
print(f'Columns: {list(df.columns)}')
print()
print('--- First 5 rows ---')
print(df.head().to_string())
print()
print('--- dtypes ---')
print(df.dtypes)
print()

df.to_csv(output_csv, index=False)
print(f'Saved to: {output_csv}')
print(f'CSV size: {Path(output_csv).stat().st_size / (1024*1024):.2f} MB')
" "$LOG_FILE" "$OUTPUT_CSV"

echo ""
echo "Done!"

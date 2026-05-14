"""
Decode an inference-log CSV directly to another CSV using the caller-supplied
schema. No Spark required; pure-Python (csv + json + base64 + the proto decoder).

Usage:
    python decode_csv_to_csv.py [input.csv] [output.csv]

Defaults:
    input  = /Users/dheerajchouhan/Downloads/test_new.csv
    output = /tmp/decoded_test_new.csv
"""

import sys

from inference_logging_client import decode_mplog_proto_csv

# Import the full 256-feature schema from the sibling script.
from decode_single_row import SCHEMA


DEFAULT_INPUT = "/Users/dheerajchouhan/Downloads/test_new.csv"
DEFAULT_OUTPUT = "/tmp/decoded_test_new.csv"


def main():
    input_csv = sys.argv[1] if len(sys.argv) > 1 else DEFAULT_INPUT
    output_csv = sys.argv[2] if len(sys.argv) > 2 else DEFAULT_OUTPUT

    print(f"input  : {input_csv}")
    print(f"output : {output_csv}")
    print(f"schema : {len(SCHEMA['data'])} features")

    n = decode_mplog_proto_csv(
        input_csv=input_csv,
        output_csv=output_csv,
        schema=SCHEMA,
    )

    print(f"decoded rows written: {n}")


if __name__ == "__main__":
    main()

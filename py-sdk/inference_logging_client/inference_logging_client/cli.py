"""Command-line interface for inference-logging-client."""

import os
import sys
import base64
import argparse

import pandas as pd

from . import decode_mplog, get_mplog_metadata, get_format_name, format_dataframe_floats
from .types import Format


def main():
    """Main entry point with CLI interface."""
    parser = argparse.ArgumentParser(
        description="Decode MPLog feature logs from proto, arrow, or parquet format",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Decode MPLog with auto-detection of format from metadata
  inference-logging-client --model-proxy-id my-model --version 1 input.bin
  
  # Decode a proto-encoded MPLog file (explicit format)
  inference-logging-client --model-proxy-id my-model --version 1 --format proto input.bin
  
  # Decode base64-encoded Arrow format from stdin
  echo "BASE64_DATA" | inference-logging-client --model-proxy-id my-model --version 1 --format arrow --base64 -
  
  # Output to CSV
  inference-logging-client --model-proxy-id my-model --version 1 input.bin -o output.csv
        """
    )
    
    parser.add_argument("input", help="Input file containing MPLog bytes (or - for stdin)")
    parser.add_argument("--model-proxy-id", "-m", required=True,
                        help="Model proxy config ID")
    parser.add_argument("--version", "-v", type=int, required=True,
                        help="Schema version")
    parser.add_argument("--format", "-f", choices=["proto", "arrow", "parquet", "auto"],
                        default="auto",
                        help="Encoding format (default: auto-detect from metadata)")
    parser.add_argument("--inference-host", default=None,
                        help="Inference service host URL (default: reads from INFERENCE_HOST env var or http://localhost:8082)")
    parser.add_argument("--hex", action="store_true",
                        help="Input is hex-encoded string")
    parser.add_argument("--base64", action="store_true",
                        help="Input is base64-encoded string")
    parser.add_argument("--no-decompress", action="store_true",
                        help="Skip automatic zstd decompression")
    parser.add_argument("--output", "-o",
                        help="Output file (CSV format, default: print to stdout)")
    parser.add_argument("--json", action="store_true",
                        help="Output as JSON instead of CSV")
    
    args = parser.parse_args()
    
    # Read input
    if args.input == "-":
        data = sys.stdin.buffer.read()
    else:
        with open(args.input, "rb") as f:
            data = f.read()
    
    # Decode input format
    if args.hex:
        try:
            data = bytes.fromhex(data.decode('utf-8').strip())
        except ValueError as e:
            print(f"Error: Invalid hex input: {e}", file=sys.stderr)
            sys.exit(1)
    elif args.base64:
        try:
            data = base64.b64decode(data)
        except Exception as e:
            print(f"Error: Invalid base64 input: {e}", file=sys.stderr)
            sys.exit(1)
    
    # Parse format (None for auto-detection)
    format_type = None if args.format == "auto" else Format(args.format)
    
    # Get inference host from argument or environment variable
    inference_host = args.inference_host or os.getenv("INFERENCE_HOST", "http://localhost:8082")
    
    # Decode MPLog
    try:
        df = decode_mplog(
            log_data=data,
            model_proxy_id=args.model_proxy_id,
            version=args.version,
            format_type=format_type,
            inference_host=inference_host,
            decompress=not args.no_decompress
        )
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    
    # Format floats before output
    df = format_dataframe_floats(df)
    
    # Output
    if args.output:
        if args.json:
            df.to_json(args.output, orient="records", indent=2)
        else:
            df.to_csv(args.output, index=False)
        print(f"Output written to {args.output}")
    else:
        if args.json:
            print(df.to_json(orient="records", indent=2))
        else:
            # Pretty print for terminal
            pd.set_option('display.max_columns', None)
            pd.set_option('display.width', None)
            pd.set_option('display.max_colwidth', 50)
            print(df.to_string(index=False))
    
    # Get metadata for summary
    metadata = get_mplog_metadata(data, decompress=not args.no_decompress)
    detected_format_name = get_format_name(metadata.format_type)
    
    # Print summary
    print(f"\n--- Summary ---", file=sys.stderr)
    print(f"Format: {detected_format_name} (from metadata)" if args.format == "auto" else f"Format: {args.format}", file=sys.stderr)
    print(f"Version: {metadata.version}", file=sys.stderr)
    print(f"Compression: {'enabled' if metadata.compression_enabled else 'disabled'}", file=sys.stderr)
    print(f"Rows: {len(df)}", file=sys.stderr)
    print(f"Columns: {len(df.columns)}", file=sys.stderr)
    print(f"Features: {', '.join(df.columns[1:5])}{'...' if len(df.columns) > 5 else ''}", file=sys.stderr)


if __name__ == "__main__":
    main()

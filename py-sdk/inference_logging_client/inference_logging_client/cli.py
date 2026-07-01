"""Command-line interface for inference-logging-client."""

import argparse
import base64
import glob
import os
import shutil
import sys
import tempfile

from . import (
    analyze_log_file,
    decode_log_file,
    decode_log_file_to_csv,
    decode_log_file_to_jsonl,
    decode_log_file_to_pandas,
    decode_mplog,
    format_dataframe_floats,
    get_format_name,
    get_mplog_metadata,
    print_analysis,
    write_parsed_log,
)
from .types import Format


def _is_log_input(path: str) -> bool:
    """Return True when ``path`` should be routed through the .log reader."""
    if path.startswith("gs://"):
        return True
    return path.lower().endswith(".log")


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
        """,
    )

    parser.add_argument(
        "input",
        help=(
            "Input source: path to MPLog bytes file, path to an asyncloguploader "
            ".log file, a gs://bucket/key URI, or '-' for stdin (MPLog bytes only)"
        ),
    )
    parser.add_argument(
        "--model-proxy-id",
        "-m",
        default=None,
        help="Model proxy config ID (required for raw MPLog input; ignored for .log/GCS)",
    )
    parser.add_argument(
        "--version",
        "-v",
        type=int,
        default=None,
        help="Schema version (required for raw MPLog input; ignored for .log/GCS)",
    )
    parser.add_argument(
        "--strict",
        action="store_true",
        help="For .log inputs: raise on any malformed frame instead of skipping",
    )
    parser.add_argument(
        "--output-format",
        choices=["spark", "pandas", "csv", "jsonl", "text", "analyze"],
        default=None,
        help=(
            "For .log / gs:// inputs, choose the output sink: 'spark' (default, "
            "prints a Spark DataFrame), 'pandas' (CSV-style print of a pandas "
            "DataFrame), 'csv' / 'jsonl' (write to --output), 'text' "
            "(asynclogparser .parsed.log format, requires --output), or "
            "'analyze' (structural report, no decoding)"
        ),
    )
    parser.add_argument(
        "--format",
        "-f",
        choices=["proto", "arrow", "parquet", "auto"],
        default="auto",
        help="Encoding format (default: auto-detect from metadata)",
    )
    parser.add_argument(
        "--inference-host",
        default=None,
        help="Inference service host URL (default: reads from INFERENCE_HOST env var or http://localhost:8082)",
    )
    parser.add_argument("--hex", action="store_true", help="Input is hex-encoded string")
    parser.add_argument("--base64", action="store_true", help="Input is base64-encoded string")
    parser.add_argument(
        "--no-decompress", action="store_true", help="Skip automatic zstd decompression"
    )
    parser.add_argument("--output", "-o", help="Output file (CSV format, default: print to stdout)")
    parser.add_argument("--json", action="store_true", help="Output as JSON instead of CSV")
    parser.add_argument(
        "--spark-master",
        default="local[*]",
        help="Spark master URL (default: local[*])",
    )

    args = parser.parse_args()

    log_mode = _is_log_input(args.input)
    out_fmt = args.output_format or ("spark" if log_mode else None)
    if out_fmt and not log_mode and out_fmt != "spark":
        print(
            f"Error: --output-format={out_fmt} only applies to .log / gs:// inputs",
            file=sys.stderr,
        )
        sys.exit(2)

    inference_host = args.inference_host or os.getenv("INFERENCE_HOST", "http://localhost:8082")

    # Non-Spark .log sinks short-circuit before Spark startup.
    if log_mode and out_fmt in {"pandas", "csv", "jsonl", "text", "analyze"}:
        try:
            if out_fmt == "analyze":
                print_analysis(analyze_log_file(args.input))
                return
            if out_fmt == "pandas":
                pdf = decode_log_file_to_pandas(
                    args.input,
                    inference_host=inference_host,
                    decompress=not args.no_decompress,
                    strict=args.strict,
                )
                if args.output:
                    pdf.to_csv(args.output, index=False)
                    print(f"Output written to {args.output} ({len(pdf)} rows)")
                else:
                    print(pdf.to_csv(index=False))
                return
            if out_fmt in {"csv", "jsonl", "text"}:
                if not args.output:
                    print(
                        f"Error: --output-format={out_fmt} requires --output PATH",
                        file=sys.stderr,
                    )
                    sys.exit(2)
                sink = {
                    "csv": decode_log_file_to_csv,
                    "jsonl": decode_log_file_to_jsonl,
                    "text": write_parsed_log,
                }[out_fmt]
                n = sink(
                    args.input,
                    args.output,
                    inference_host=inference_host,
                    decompress=not args.no_decompress,
                    strict=args.strict,
                )
                print(f"Output written to {args.output} ({n} {'records' if out_fmt=='text' else 'rows'})")
                return
        except Exception as e:
            print(f"Error: {e}", file=sys.stderr)
            sys.exit(1)

    data: bytes = b""

    if not log_mode:
        # Raw MPLog bytes path — requires schema identifiers.
        if args.model_proxy_id is None or args.version is None:
            print(
                "Error: --model-proxy-id and --version are required when input is "
                "MPLog bytes (anything other than a .log file or gs:// URI)",
                file=sys.stderr,
            )
            sys.exit(2)

        if args.input == "-":
            data = sys.stdin.buffer.read()
        else:
            with open(args.input, "rb") as f:
                data = f.read()

        if args.hex:
            try:
                data = bytes.fromhex(data.decode("utf-8").strip())
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

    # Create SparkSession for CLI
    from pyspark.sql import SparkSession

    spark = SparkSession.builder \
        .appName("inference-logging-client") \
        .master(args.spark_master) \
        .config("spark.ui.showConsoleProgress", "false") \
        .config("spark.driver.memory", "2g") \
        .getOrCreate()

    # Suppress Spark logging for cleaner CLI output
    spark.sparkContext.setLogLevel("ERROR")

    try:
        if log_mode:
            df = decode_log_file(
                source=args.input,
                spark=spark,
                inference_host=inference_host,
                decompress=not args.no_decompress,
                strict=args.strict,
            )
        else:
            df = decode_mplog(
                log_data=data,
                model_proxy_id=args.model_proxy_id,
                version=args.version,
                spark=spark,
                format_type=format_type,
                inference_host=inference_host,
                decompress=not args.no_decompress,
            )

        # Format floats before output
        df = format_dataframe_floats(df)

        # Output
        if args.output:
            if args.json:
                # Write as JSON
                df.coalesce(1).write.mode("overwrite").json(args.output)
            else:
                # Write as CSV
                df.coalesce(1).write.mode("overwrite").option("header", "true").csv(args.output)
            print(f"Output written to {args.output}")
        else:
            if args.json:
                # Avoid collect() for large DataFrames: write to temp dir then stream to stdout
                tmpdir = tempfile.mkdtemp(prefix="inference_logging_client_json_")
                try:
                    df.coalesce(1).write.mode("overwrite").json(tmpdir)
                    part_files = sorted(glob.glob(os.path.join(tmpdir, "part-*")))
                    print("[")
                    first = True
                    for path in part_files:
                        with open(path) as f:
                            for line in f:
                                line = line.strip()
                                if line:
                                    if not first:
                                        print(",")
                                    print("  " + line, end="")
                                    first = False
                    print("\n]" if not first else "]")
                finally:
                    shutil.rmtree(tmpdir, ignore_errors=True)
            else:
                # Show table (only fetches 20 rows, no full collect)
                df.show(truncate=False)

        # Print summary
        print("\n--- Summary ---", file=sys.stderr)
        if log_mode:
            print(f"Source: {args.input} (.log mode)", file=sys.stderr)
        else:
            # Get metadata for summary (raw-MPLog mode only)
            metadata = get_mplog_metadata(data, decompress=not args.no_decompress)
            detected_format_name = get_format_name(metadata.format_type)
            print(
                f"Format: {detected_format_name} (from metadata)"
                if args.format == "auto"
                else f"Format: {args.format}",
                file=sys.stderr,
            )
            print(f"Version: {metadata.version}", file=sys.stderr)
            print(
                f"Compression: {'enabled' if metadata.compression_enabled else 'disabled'}",
                file=sys.stderr,
            )
        # Avoid full count() for huge DataFrames: use limit(1).count() for empty check only
        try:
            row_count = df.count()
            print(f"Rows: {row_count}", file=sys.stderr)
        except Exception:
            print("Rows: (count skipped - use --output to write without summary)", file=sys.stderr)
        print(f"Columns: {len(df.columns)}", file=sys.stderr)
        col_preview = df.columns[1:5] if len(df.columns) > 1 else []
        print(
            f"Features: {', '.join(col_preview)}{'...' if len(df.columns) > 5 else ''}",
            file=sys.stderr,
        )
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    finally:
        # Stop SparkSession
        spark.stop()


if __name__ == "__main__":
    main()

"""
bharatml -- CLI for the Feature Computation Platform.

CI commands (no network required):
  bharatml manifest         Extract specs and generate manifest.json
  bharatml diff             Compare two manifests locally

Internal commands (require horizon access):
  bharatml apply            Register manifest with horizon + generate DAGs
  bharatml plan             Compute full impact analysis via horizon
  bharatml lineage          View asset dependency graph
  bharatml serving          Toggle serving on/off
  bharatml necessity        View current necessity states
  bharatml backfill         Run explicit backfill
"""
from __future__ import annotations

import argparse
import json
import logging
import subprocess
import sys


def get_git_sha() -> str:
    """Get current git SHA."""
    try:
        result = subprocess.run(
            ["git", "rev-parse", "HEAD"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout.strip()
    except Exception:
        return "unknown"


def get_git_ref() -> str:
    """Get current git branch/ref."""
    try:
        result = subprocess.run(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout.strip()
    except Exception:
        return "unknown"


def _build_parser() -> argparse.ArgumentParser:
    """Build the CLI argument parser. Separated for testability."""
    parser = argparse.ArgumentParser(
        prog="bharatml",
        description="Feature Computation Platform CLI",
    )
    parser.add_argument("--verbose", "-v", action="store_true")
    subparsers = parser.add_subparsers(dest="command")

    # -- manifest (CI-safe, no network) --
    manifest_parser = subparsers.add_parser(
        "manifest",
        help="Extract AssetSpecs and generate .bharatml/manifest.json",
    )
    manifest_parser.add_argument(
        "--notebook-dir",
        default="./notebooks",
        help="Directory containing notebook .py files",
    )
    manifest_parser.add_argument(
        "--output-dir",
        default=".",
        help="Where to write .bharatml/ directory (default: repo root)",
    )
    manifest_parser.add_argument(
        "--git-sha",
        default=None,
        help="Git SHA to stamp on assets (auto-detected if not provided)",
    )

    # -- diff (CI-safe, no network) --
    diff_parser = subparsers.add_parser(
        "diff",
        help="Compare current manifest against a base manifest",
    )
    diff_parser.add_argument(
        "--base", required=True, help="Path to base manifest.json"
    )
    diff_parser.add_argument(
        "--proposed", required=True, help="Path to proposed manifest.json"
    )
    diff_parser.add_argument(
        "--format",
        choices=["text", "json", "markdown"],
        default="text",
        dest="output_format",
    )

    # -- apply (internal network only) --
    apply_parser = subparsers.add_parser(
        "apply",
        help="Register manifest with horizon (internal network only)",
    )
    apply_parser.add_argument(
        "--manifest",
        default=".bharatml/manifest.json",
        help="Path to manifest.json",
    )
    apply_parser.add_argument("--horizon-url", required=True)
    apply_parser.add_argument("--horizon-api-key", default=None)
    apply_parser.add_argument("--dag-output-dir", default=None)

    # -- plan (internal network only) --
    plan_parser = subparsers.add_parser(
        "plan",
        help="Compute full impact analysis via horizon (internal network only)",
    )
    plan_parser.add_argument(
        "--manifest",
        default=".bharatml/manifest.json",
    )
    plan_parser.add_argument("--horizon-url", required=True)
    plan_parser.add_argument("--horizon-api-key", default=None)
    plan_parser.add_argument(
        "--format",
        choices=["text", "json", "markdown"],
        default="text",
        dest="output_format",
    )

    # -- lineage (internal) --
    lineage_parser = subparsers.add_parser("lineage", help="View lineage graph")
    lineage_parser.add_argument("asset_name", nargs="?")
    lineage_parser.add_argument("--downstream", action="store_true")
    lineage_parser.add_argument("--upstream", action="store_true")
    lineage_parser.add_argument("--horizon-url", required=True)

    # -- serving (internal) --
    serving_parser = subparsers.add_parser("serving", help="Toggle serving")
    serving_sub = serving_parser.add_subparsers(dest="serving_action")
    set_parser = serving_sub.add_parser("set")
    set_parser.add_argument("asset_name")
    set_parser.add_argument(
        "--serving",
        type=lambda x: x.lower() == "true",
        required=True,
    )
    set_parser.add_argument("--reason", required=True)
    set_parser.add_argument("--by", required=True)
    set_parser.add_argument("--horizon-url", required=True)

    # -- necessity (internal) --
    nec_parser = subparsers.add_parser("necessity", help="View necessity states")
    nec_parser.add_argument("--horizon-url", required=True)

    # -- backfill (internal) --
    backfill_parser = subparsers.add_parser("backfill", help="Run explicit backfill")
    backfill_parser.add_argument("asset_name")
    backfill_parser.add_argument(
        "--range", required=True, help="e.g. 2026-01-01..2026-02-01"
    )
    backfill_parser.add_argument("--horizon-url", required=True)

    return parser


def main(argv: list | None = None) -> None:
    parser = _build_parser()
    args = parser.parse_args(argv)

    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(message)s",
    )

    if args.command == "manifest":
        _cmd_manifest(args)
    elif args.command == "diff":
        _cmd_diff(args)
    elif args.command == "apply":
        _cmd_apply(args)
    elif args.command == "plan":
        _cmd_plan(args)
    elif args.command == "lineage":
        _cmd_lineage(args)
    elif args.command == "serving":
        _cmd_serving(args)
    elif args.command == "necessity":
        _cmd_necessity(args)
    elif args.command == "backfill":
        _cmd_backfill(args)
    else:
        parser.print_help()
        sys.exit(1)


def _cmd_manifest(args: argparse.Namespace) -> None:
    from .manifest import generate_manifest

    git_sha = args.git_sha or get_git_sha()
    git_ref = get_git_ref()
    manifest = generate_manifest(
        notebook_dir=args.notebook_dir,
        output_dir=args.output_dir,
        git_sha=git_sha,
        git_ref=git_ref,
    )
    summary = manifest["summary"]
    print(f"Manifest generated: {summary['total_assets']} assets")
    print(f"  Entities: {summary['by_entity']}")
    print(f"  Triggers: {summary['by_trigger']}")
    print(
        f"  Serving: {summary['serving_count']} active, "
        f"{summary['non_serving_count']} non-serving"
    )


def _cmd_diff(args: argparse.Namespace) -> None:
    from .manifest import diff_manifests, read_manifest

    base = read_manifest(args.base)
    proposed = read_manifest(args.proposed)
    diff = diff_manifests(base, proposed)

    if args.output_format == "json":
        print(json.dumps(diff, indent=2))
    else:
        if not diff["has_changes"]:
            print("No changes detected.")
        else:
            if diff["added"]:
                print(f"NEW:      {', '.join(diff['added'])}")
            if diff["modified"]:
                for m in diff["modified"]:
                    print(f"MODIFIED: {m['name']} ({', '.join(m['changes'])})")
            if diff["removed"]:
                print(f"REMOVED:  {', '.join(diff['removed'])}")


def _cmd_apply(args: argparse.Namespace) -> None:
    from .apply import apply_manifest

    result = apply_manifest(
        manifest_path=args.manifest,
        horizon_url=args.horizon_url,
        dag_output_dir=args.dag_output_dir,
        horizon_api_key=args.horizon_api_key,
    )
    print(f"Registered {result['assets_registered']} assets")
    n = result["necessity"]
    print(
        f"Necessity: {n['active']} ACTIVE, "
        f"{n['transient']} TRANSIENT, "
        f"{n['skipped']} SKIPPED"
    )
    if result["dag_files"]:
        print(f"Generated {len(result['dag_files'])} DAG files")


def _cmd_plan(args: argparse.Namespace) -> None:
    from .apply import plan_from_manifest

    plan = plan_from_manifest(
        manifest_path=args.manifest,
        horizon_url=args.horizon_url,
        horizon_api_key=args.horizon_api_key,
    )
    if args.output_format == "json":
        print(json.dumps(plan, indent=2))
    else:
        changes = plan.get("changes", [])
        if not changes:
            print("No changes vs production.")
        else:
            symbols = {"added": "+", "modified": "~", "removed": "-"}
            for c in changes:
                symbol = symbols.get(c["change_type"], "?")
                print(f"  {symbol} {c['asset_name']}")
            total = plan.get("total_partitions", 0)
            if total:
                print(f"\nRecomputation: {total} partitions")
            cost = plan.get("total_estimated_cost_usd")
            if cost:
                print(f"Estimated cost: ${cost:.2f}")


def _cmd_lineage(args: argparse.Namespace) -> None:
    from feature_compute_engine.client.horizon_client import (
        HorizonClient,
        HorizonClientConfig,
    )

    client = HorizonClient(HorizonClientConfig(base_url=args.horizon_url))
    data = client.get_lineage(args.asset_name)
    print(json.dumps(data, indent=2))


def _cmd_serving(args: argparse.Namespace) -> None:
    if args.serving_action == "set":
        from feature_compute_engine.client.horizon_client import (
            HorizonClient,
            HorizonClientConfig,
        )

        client = HorizonClient(
            HorizonClientConfig(base_url=args.horizon_url)
        )
        client.set_serving_override(
            args.asset_name, args.serving, args.reason, args.by
        )
        state = "ON" if args.serving else "OFF"
        print(f"Serving {state} for {args.asset_name}")
    else:
        print("Usage: bharatml serving set <asset> --serving true/false --reason ... --by ...")
        sys.exit(1)


def _cmd_necessity(args: argparse.Namespace) -> None:
    from feature_compute_engine.client.horizon_client import (
        HorizonClient,
        HorizonClientConfig,
    )

    client = HorizonClient(HorizonClientConfig(base_url=args.horizon_url))
    data = client.get_necessity()
    for asset_name, necessity in sorted(data.items()):
        print(f"  {asset_name:40s} {necessity}")


def _cmd_backfill(args: argparse.Namespace) -> None:
    from feature_compute_engine.client.horizon_client import (
        HorizonClient,
        HorizonClientConfig,
    )

    client = HorizonClient(HorizonClientConfig(base_url=args.horizon_url))
    data = client._request(
        "POST",
        f"/assets/{args.asset_name}/backfill",
        json={"range": getattr(args, "range")},
    )
    print(json.dumps(data, indent=2))


if __name__ == "__main__":
    main()

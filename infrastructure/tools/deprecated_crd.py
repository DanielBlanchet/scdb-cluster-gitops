#!/usr/bin/env python3

import argparse
import csv
import json
import re
import subprocess
import sys
from typing import Any


def run_kubectl(args: list[str]) -> str:
    result = subprocess.run(args, capture_output=True, text=True)
    if result.returncode != 0:
        err = result.stderr.strip() or "Unknown kubectl error"
        raise RuntimeError(err)
    return result.stdout


def load_kubeconfig() -> dict[str, Any]:
    output = run_kubectl(["kubectl", "config", "view", "-o", "json"])
    try:
        return json.loads(output)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"Invalid JSON returned by kubectl config view: {exc}") from exc


def get_contexts(selected_name: str | None, kubeconfig: dict[str, Any]) -> list[str]:
    if selected_name:
        # Preferred lookup: by cluster name (--cluster)
        contexts_for_cluster: list[str] = []
        for context_entry in kubeconfig.get("contexts", []):
            context_name = context_entry.get("name", "").strip()
            cluster_name = context_entry.get("context", {}).get("cluster", "").strip()
            if cluster_name == selected_name and context_name:
                contexts_for_cluster.append(context_name)

        if contexts_for_cluster:
            preferred_context = next((ctx for ctx in contexts_for_cluster if ctx == selected_name), contexts_for_cluster[0])
            return [preferred_context]

        # Backward compatibility: allow legacy --context values.
        for context_entry in kubeconfig.get("contexts", []):
            context_name = context_entry.get("name", "").strip()
            if context_name == selected_name:
                return [context_name]

        raise RuntimeError(f"Cluster or context '{selected_name}' not found in kubeconfig.")

    cluster_to_contexts: dict[str, list[str]] = {}
    cluster_order: list[str] = []

    for context_entry in kubeconfig.get("contexts", []):
        context_name = context_entry.get("name", "").strip()
        cluster_name = context_entry.get("context", {}).get("cluster", "").strip()

        if not context_name:
            continue

        cluster_key = cluster_name or context_name
        if cluster_key not in cluster_to_contexts:
            cluster_to_contexts[cluster_key] = []
            cluster_order.append(cluster_key)
        if context_name not in cluster_to_contexts[cluster_key]:
            cluster_to_contexts[cluster_key].append(context_name)

    selected_contexts: list[str] = []
    for cluster_key in cluster_order:
        contexts_for_cluster = cluster_to_contexts[cluster_key]
        preferred_context = next(
            (ctx for ctx in contexts_for_cluster if ctx == cluster_key),
            contexts_for_cluster[0],
        )
        selected_contexts.append(preferred_context)

    return selected_contexts


def get_crds(context: str) -> list[dict[str, Any]]:
    output = run_kubectl(["kubectl", "--context", context, "get", "crd", "-o", "json"])
    try:
        data = json.loads(output)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"Invalid JSON returned by kubectl for context '{context}': {exc}") from exc

    return data.get("items", [])


def get_context_cluster_map(kubeconfig: dict[str, Any]) -> dict[str, str]:
    mapping: dict[str, str] = {}
    for context_entry in kubeconfig.get("contexts", []):
        context_name = context_entry.get("name", "").strip()
        cluster_name = context_entry.get("context", {}).get("cluster", "").strip()
        if context_name:
            mapping[context_name] = cluster_name or context_name
    return mapping


def list_clusters(kubeconfig: dict[str, Any]) -> list[str]:
    context_cluster_map = get_context_cluster_map(kubeconfig)
    clusters = sorted({cluster for cluster in context_cluster_map.values() if cluster})
    return clusters


def collect_crd_summary(crds: list[dict[str, Any]]) -> list[tuple[str, str, str, str]]:
    rows: list[tuple[str, str, str, str]] = []

    for crd in crds:
        crd_name = crd.get("metadata", {}).get("name", "<unknown-crd>")
        versions = crd.get("spec", {}).get("versions", [])
        latest_version = "<unknown-version>"
        deprecated_rows: list[tuple[str, str]] = []

        if versions:
            latest_version = versions[0].get("name", "<unknown-version>")

        for version in versions:
            version_name = version.get("name", "<unknown-version>")
            if version.get("storage") is True:
                latest_version = version_name
            if version.get("deprecated") is True:
                reason = version.get("deprecationWarning", "").strip()
                reason_text = reason if reason else "no deprecationWarning"
                deprecated_rows.append((version_name, reason_text))

        if deprecated_rows:
            for deprecated_version, reason_text in deprecated_rows:
                rows.append((crd_name, latest_version, deprecated_version, reason_text))
        else:
            rows.append((crd_name, latest_version, "-", "-"))

    rows.sort(key=lambda row: row[0])
    return rows


def print_versions_table(rows: list[tuple[str, str, str, str]]) -> None:
    if not rows:
        print("No CRDs found.")
        return

    headers = ("CRDs", "Latest", "Deprecated", "Reason")
    all_rows = [headers, *rows]

    crd_width = max(len(r[0]) for r in all_rows)
    version_width = max(len(r[1]) for r in all_rows)
    deprecated_width = max(len(r[2]) for r in all_rows)
    reason_width = max(len(r[3]) for r in all_rows)

    print(
        f"{headers[0]:<{crd_width}}  {headers[1]:<{version_width}}  {headers[2]:<{deprecated_width}}  {headers[3]:<{reason_width}}"
    )
    print(f"{'-' * crd_width}  {'-' * version_width}  {'-' * deprecated_width}  {'-' * reason_width}")
    for crd_name, version_name, deprecated, reason in rows:
        print(f"{crd_name:<{crd_width}}  {version_name:<{version_width}}  {deprecated:<{deprecated_width}}  {reason:<{reason_width}}")


def safe_filename(value: str) -> str:
    normalized = re.sub(r"[^A-Za-z0-9._-]+", "-", value.strip())
    return normalized.strip("-") or "context"


def aggregate_machine_rows(machine_rows: list[dict[str, str]]) -> list[dict[str, Any]]:
    grouped: dict[tuple[str, str, str], dict[str, Any]] = {}

    for row in machine_rows:
        key = (row["Context"], row["CRDs"], row["Latest"])
        if key not in grouped:
            grouped[key] = {
                "Context": row["Context"],
                "CRDs": row["CRDs"],
                "Latest": row["Latest"],
                "Deprecated": [],
            }

        if row["Deprecated"] != "-":
            grouped[key]["Deprecated"].append(
                {
                    "version": row["Deprecated"],
                    "reason": row["Reason"],
                }
            )

    return list(grouped.values())


def main() -> int:
    parser = argparse.ArgumentParser(
        description="List deprecated CRD versions for one cluster, or one context per cluster."
    )
    parser.add_argument(
        "--cluster",
        help="Optional cluster name. If omitted, one context per cluster is processed.",
    )
    parser.add_argument("--context", help=argparse.SUPPRESS)
    parser.add_argument(
        "--deprecated-only",
        action="store_true",
        help="List only CRDs that have deprecated versions.",
    )
    parser.add_argument(
        "--output",
        choices=["table", "csv", "json"],
        default="table",
        help="Output format (default: table).",
    )
    parser.add_argument(
        "--output-file",
        help="Output file path for csv/json. If omitted, a default filename is generated from cluster.",
    )
    parser.add_argument(
        "--list-cluster",
        action="store_true",
        help="List available clusters from kubeconfig and exit.",
    )
    args = parser.parse_args()

    try:
        kubeconfig = load_kubeconfig()
    except RuntimeError as exc:
        print(f"Error while reading kubeconfig: {exc}", file=sys.stderr)
        return 1

    if args.list_cluster:
        try:
            clusters = list_clusters(kubeconfig)
        except RuntimeError as exc:
            print(f"Error while listing clusters: {exc}", file=sys.stderr)
            return 1

        if not clusters:
            print("No clusters found in kubeconfig.")
            return 0

        for cluster in clusters:
            print(cluster)
        return 0

    try:
        selected_cluster = args.cluster or args.context
        contexts = get_contexts(selected_cluster, kubeconfig)
        context_cluster_map = get_context_cluster_map(kubeconfig)
    except RuntimeError as exc:
        print(f"Error while retrieving contexts: {exc}", file=sys.stderr)
        return 1

    if not contexts:
        output_path = args.output_file or f"all-contexts.{args.output}"
        if args.output == "json":
            with open(output_path, "w", encoding="utf-8") as fp:
                fp.write("[]\n")
            print(f"Output written to {output_path}", file=sys.stderr)
        elif args.output == "csv":
            with open(output_path, "w", newline="", encoding="utf-8") as fp:
                writer = csv.DictWriter(
                    fp,
                    fieldnames=["Context", "CRDs", "Latest", "Deprecated", "Reason"],
                )
                writer.writeheader()
            print(f"Output written to {output_path}", file=sys.stderr)
        else:
            print("No kubectl contexts found.")
        return 0

    exit_code = 0
    machine_rows: list[dict[str, str]] = []

    for ctx in contexts:
        if args.output == "table":
            cluster_display_name = context_cluster_map.get(ctx, ctx)
            print("==============================")
            print(f"Cluster: {cluster_display_name}")
            print("==============================")

        try:
            crds = get_crds(ctx)
            versions = collect_crd_summary(crds)
        except RuntimeError as exc:
            print(f"Error for context '{ctx}': {exc}", file=sys.stderr)
            print(file=sys.stderr)
            exit_code = 1
            continue

        deprecated_count = sum(1 for _, _, deprecated, _ in versions if deprecated != "-")
        displayed_versions = versions
        if args.deprecated_only:
            displayed_versions = [row for row in versions if row[2] != "-"]

        if args.output == "table":
            print(
                f"CRDs: {len(versions)} | with deprecated versions: {deprecated_count} | displayed: {len(displayed_versions)}"
            )
            print_versions_table(displayed_versions)
            print()
            continue

        for crd_name, latest, deprecated, reason in displayed_versions:
            machine_rows.append(
                {
                    "Context": ctx,
                    "CRDs": crd_name,
                    "Latest": latest,
                    "Deprecated": deprecated,
                    "Reason": reason,
                }
            )

    if args.output in ("csv", "json"):
        default_base = (
            safe_filename(context_cluster_map.get(contexts[0], contexts[0])) if len(contexts) == 1 else "all-contexts"
        )
        output_path = args.output_file or f"{default_base}.{args.output}"
        aggregated_rows = aggregate_machine_rows(machine_rows)

        if args.output == "csv":
            with open(output_path, "w", newline="", encoding="utf-8") as fp:
                writer = csv.DictWriter(
                    fp,
                    fieldnames=["Context", "CRDs", "Latest", "Deprecated", "Reason"],
                )
                writer.writeheader()
                for row in aggregated_rows:
                    deprecated_entries = row["Deprecated"]
                    csv_row = {
                        "Context": row["Context"],
                        "CRDs": row["CRDs"],
                        "Latest": row["Latest"],
                        "Deprecated": "\n".join(item["version"] for item in deprecated_entries) if deprecated_entries else "-",
                        "Reason": "\n".join(item["reason"] for item in deprecated_entries) if deprecated_entries else "-",
                    }
                    writer.writerow(csv_row)
        else:
            with open(output_path, "w", encoding="utf-8") as fp:
                json.dump(aggregated_rows, fp, indent=2)
                fp.write("\n")

        print(f"Output written to {output_path}", file=sys.stderr)

    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())

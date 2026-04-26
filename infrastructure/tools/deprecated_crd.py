#!/usr/bin/env python3

import argparse
import json
import subprocess
import sys
from typing import Any


def run_kubectl(args: list[str]) -> str:
    result = subprocess.run(args, capture_output=True, text=True)
    if result.returncode != 0:
        err = result.stderr.strip() or "Unknown kubectl error"
        raise RuntimeError(err)
    return result.stdout


def get_contexts(context: str | None) -> list[str]:
    if context:
        return [context]

    output = run_kubectl(["kubectl", "config", "get-contexts", "-o", "name"])
    return [line.strip() for line in output.splitlines() if line.strip()]


def get_crds(context: str) -> list[dict[str, Any]]:
    output = run_kubectl(["kubectl", "--context", context, "get", "crd", "-o", "json"])
    try:
        data = json.loads(output)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"Invalid JSON returned by kubectl for context '{context}': {exc}") from exc

    return data.get("items", [])


def find_deprecated_versions(crds: list[dict[str, Any]]) -> list[tuple[str, str]]:
    deprecated: list[tuple[str, str]] = []

    for crd in crds:
        crd_name = crd.get("metadata", {}).get("name", "<unknown-crd>")
        versions = crd.get("spec", {}).get("versions", [])

        for version in versions:
            if version.get("deprecated") is True:
                version_name = version.get("name", "<unknown-version>")
                deprecated.append((crd_name, version_name))

    return deprecated


def main() -> int:
    parser = argparse.ArgumentParser(
        description="List deprecated CRD versions for one or all Kubernetes contexts."
    )
    parser.add_argument(
        "--context",
        help="Optional kubectl context name. If omitted, all contexts are processed.",
    )
    args = parser.parse_args()

    try:
        contexts = get_contexts(args.context)
    except RuntimeError as exc:
        print(f"Error while retrieving contexts: {exc}", file=sys.stderr)
        return 1

    if not contexts:
        print("No kubectl contexts found.")
        return 0

    exit_code = 0

    for ctx in contexts:
        print("==============================")
        print(f"Cluster: {ctx}")
        print("==============================")

        try:
            crds = get_crds(ctx)
            deprecated = find_deprecated_versions(crds)
        except RuntimeError as exc:
            print(f"Error for context '{ctx}': {exc}", file=sys.stderr)
            print()
            exit_code = 1
            continue

        if not deprecated:
            print("No deprecated CRDs ✅")
        else:
            print(f"Deprecated CRDs: {len(deprecated)}")
            for crd_name, version_name in deprecated:
                print(f"{crd_name} {version_name}")

        print()

    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())

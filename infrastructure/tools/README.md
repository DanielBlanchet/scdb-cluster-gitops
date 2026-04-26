# deprecated_crd.py

Python script to list Kubernetes CRDs and their deprecated versions.

## Prerequisites

- `python3`
- `kubectl` configured (kubeconfig + contexts)

## Usage

```bash
python3 infrastructure/tools/deprecated_crd.py [options]
```

## Options

- `--cluster <name>`: target a specific cluster (automatic selection of the associated context). By default, one context per cluster is processed (priority to a context with the same name as the cluster).
- `--deprecated-only`: show only CRDs that have at least one deprecated version.
- `--output table|csv|json`: output format (`table` by default).
- `--output-file <path>`: force output file path for `csv`/`json`.
- `--list-cluster`: list available clusters from kubeconfig (with Kubernetes version) and exit.

## Outputs

### table (default)

Console output in columns:

- `CRDs`
- `Latest`
- `Deprecated`
- `Reason`

If a CRD has multiple deprecated versions, it appears on multiple lines.

### csv

- 1 row per CRD (aggregated output)
- columns: `Cluster, CRDs, Latest, Deprecated, Reason`
- if there are multiple deprecated versions, they are kept in the same cell, separated by newline
- default filename:
  - single targeted cluster: `<cluster>.csv`
  - multiple clusters: `all-contexts.csv`

### json

- 1 object per CRD (aggregated output)
- format:

```json
{
  "Cluster": "...",
  "CRDs": "...",
  "Latest": "...",
  "Deprecated": [
    {"version": "v1beta1", "reason": "..."}
  ]
}
```

- default filename:
  - single targeted cluster: `<cluster>.json`
  - multiple clusters: `all-contexts.json`

## Examples

```bash
# Table output (one context per cluster)
python3 infrastructure/tools/deprecated_crd.py

# List available clusters
python3 infrastructure/tools/deprecated_crd.py --list-cluster

# Single cluster
python3 infrastructure/tools/deprecated_crd.py --cluster prod

# Deprecated CRDs only
python3 infrastructure/tools/deprecated_crd.py --deprecated-only

# CSV export
python3 infrastructure/tools/deprecated_crd.py --cluster prod --output csv

# JSON export with explicit output path
python3 infrastructure/tools/deprecated_crd.py --cluster prod --output json --output-file /tmp/crds-prod.json
```

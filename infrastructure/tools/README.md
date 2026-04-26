# deprecated_crd.py

Script Python pour lister les CRDs Kubernetes et leurs versions depreciees.

## Prerequis

- `python3`
- `kubectl` configure (kubeconfig + contextes)

## Usage

```bash
python3 infrastructure/tools/deprecated_crd.py [options]
```

## Options

- `--cluster <nom>`: cible un cluster specifique (selection automatique du contexte associe). Par defaut, un seul contexte par cluster est traite (priorite au contexte qui a le meme nom que le cluster).
- `--deprecated-only`: affiche uniquement les CRDs ayant au moins une version depreciee.
- `--output table|csv|json`: format de sortie (`table` par defaut).
- `--output-file <chemin>`: force le fichier de sortie pour `csv`/`json`.
- `--list-cluster`: liste les clusters disponibles dans le kubeconfig puis quitte.

## Sorties

### table (par defaut)

Affichage console en colonnes:

- `CRDs`
- `Latest`
- `Deprecated`
- `Reason`

Si un CRD a plusieurs versions depreciees, il apparait sur plusieurs lignes.

### csv

- 1 ligne par CRD (sortie agregee)
- colonnes: `Context, CRDs, Latest, Deprecated, Reason`
- si plusieurs versions depreciees, elles sont dans la meme cellule, separees par un retour a la ligne
- fichier par defaut:
  - un seul cluster cible: `<cluster>.csv`
  - plusieurs clusters: `all-contexts.csv`

### json

- 1 objet par CRD (sortie agregee)
- format:

```json
{
  "Context": "...",
  "CRDs": "...",
  "Latest": "...",
  "Deprecated": [
    {"version": "v1beta1", "reason": "..."}
  ]
}
```

- fichier par defaut:
  - un seul cluster cible: `<cluster>.json`
  - plusieurs clusters: `all-contexts.json`

## Exemples

```bash
# Affichage table (un contexte par cluster)
python3 infrastructure/tools/deprecated_crd.py

# Lister les clusters disponibles
python3 infrastructure/tools/deprecated_crd.py --list-cluster

# Un seul cluster
python3 infrastructure/tools/deprecated_crd.py --cluster prod

# Uniquement les CRDs depreciees
python3 infrastructure/tools/deprecated_crd.py --deprecated-only

# Export CSV
python3 infrastructure/tools/deprecated_crd.py --cluster prod --output csv

# Export JSON avec chemin force
python3 infrastructure/tools/deprecated_crd.py --cluster prod --output json --output-file /tmp/crds-prod.json
```

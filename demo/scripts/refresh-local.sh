#!/bin/sh
set -eu
DEMO_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$DEMO_ROOT/backend"
# Read the local public astronomy database; no migration or write is performed.
go run ./cmd/export -out ../data
cd "$DEMO_ROOT"
python3 scripts/prepare-assets.py
printf 'Local snapshots ready. Build and upload the bundle to publish them.\n'

#!/bin/sh
set -eu
DEMO_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$DEMO_ROOT/frontend"
pnpm build
cd "$DEMO_ROOT/backend"
mkdir -p ../bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o ../bin/aurora-demo ./cmd/demo
cd "$DEMO_ROOT"
mkdir -p release/web/aurora release/bin release/data release/deploy
rsync -a --delete frontend/dist/ release/web/aurora/
cp bin/aurora-demo release/bin/
cp data/*.json data/*.avnl release/data/
# Remote metadata is only useful on the computer; never ship remote image URLs.
rm -f release/data/image-wall-remote.json release/data/weather-shanghai.json
cp deploy/* release/deploy/
COPYFILE_DISABLE=1 tar --no-xattrs -czf aurora-demo-linux-amd64.tar.gz -C release .
printf 'Bundle: %s/aurora-demo-linux-amd64.tar.gz\n' "$DEMO_ROOT"

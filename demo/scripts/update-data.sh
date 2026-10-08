#!/bin/bash
set -euo pipefail
DEMO_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
stage=$(mktemp -d /private/tmp/aurora-data.XXXXXX)
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/data" "$stage/web/aurora"
cp "$DEMO_ROOT"/data/*.json "$stage/data/"
rm -f "$stage/data/image-wall-remote.json" "$stage/data/weather-shanghai.json"
cp -R "$DEMO_ROOT/frontend/public/demo-images" "$stage/web/aurora/"
COPYFILE_DISABLE=1 tar --no-xattrs -czf "$stage/update.tar.gz" -C "$stage" data web
scp "$stage/update.tar.gz" tencent:/tmp/aurora-demo-data-update.tar.gz
scp "$DEMO_ROOT/deploy/update-server.sh" tencent:/tmp/aurora-demo-update.sh
ssh tencent 'sudo -n bash /tmp/aurora-demo-update.sh /tmp/aurora-demo-data-update.tar.gz'

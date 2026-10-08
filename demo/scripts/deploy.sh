#!/bin/sh
set -eu
DEMO_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
# This initial migration removes CSTS containers and legacy services, retaining
# its recovery volumes. /resume/ and its TLS configuration stay unchanged.
scp "$DEMO_ROOT/aurora-demo-linux-amd64.tar.gz" tencent:/tmp/aurora-demo-linux-amd64.tar.gz
scp "$DEMO_ROOT/deploy/install-server.sh" tencent:/tmp/aurora-demo-install.sh
ssh tencent 'sudo -n bash /tmp/aurora-demo-install.sh /tmp/aurora-demo-linux-amd64.tar.gz'

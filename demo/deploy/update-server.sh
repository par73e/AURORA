#!/bin/bash
set -euo pipefail
bundle=${1:?bundle required}
base=/opt/aurora-demo
old=$(readlink -f "$base/current")
case "$old" in "$base"/releases/*) ;; *) echo 'Unexpected current release path'; exit 1;; esac
next=$base/releases/$(date -u +%Y%m%dT%H%M%SZ)-data
cp -a "$old" "$next"
tar -xzf "$bundle" -C "$next"
chmod -R a+rX "$next"
ln -s "$next" "$base/current.next"
mv -Tf "$base/current.next" "$base/current"
rollback() {
    ln -sfn "$old" "$base/current.rollback"
    mv -Tf "$base/current.rollback" "$base/current"
    systemctl restart aurora-demo
    echo 'New data failed validation; previous release restored.' >&2
}
if ! systemctl restart aurora-demo; then rollback; exit 1; fi
for attempt in {1..15}; do
    if curl -fsS http://127.0.0.1:18082/api/health; then
        printf '\nData release %s ready.\n' "$next"
        exit 0
    fi
    sleep 1
done
rollback
exit 1

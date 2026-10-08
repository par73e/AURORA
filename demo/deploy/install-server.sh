#!/bin/bash
# Initial migration is authorized to retire only the csts-docker Compose project
# and four specifically identified legacy CSTS services. No global Docker prune.
set -euo pipefail
bundle=${1:?bundle path required}
release_id=$(date -u +%Y%m%dT%H%M%SZ)
base=/opt/aurora-demo
release=$base/releases/$release_id
backup=$base/backups/$release_id
mkdir -p "$release" "$backup"
find /home/ubuntu/Project/resume -type f -print0 | sort -z | xargs -0 sha256sum > "$backup/resume.before.sha256"
cp -a /etc/nginx/sites-available/csts "$backup/nginx.csts" 2>/dev/null || true
cp -a /etc/nginx/sites-available/aurora-demo "$backup/nginx.aurora" 2>/dev/null || true
for unit in csts-gateway csts-product csts-trade csts-user; do
    if [ -f "/etc/systemd/system/$unit.service" ]; then cp -a "/etc/systemd/system/$unit.service" "$backup/"; fi
done
tar -xzf "$bundle" -C "$release"
chmod -R a+rX "$release"
chmod 755 "$release/bin/aurora-demo"
# Validate the candidate server blocks before changing the active configuration.
{ printf 'events {}\nhttp {\ninclude /etc/nginx/mime.types;\n'; cat "$release/deploy/aurora.nginx.conf"; printf '\n}\n'; } > "$backup/nginx.candidate.conf"
nginx -t -c "$backup/nginx.candidate.conf"
ln -s "$release" "$base/current.next"
mv -Tf "$base/current.next" "$base/current"
install -m 644 "$release/deploy/aurora-demo.service" /etc/systemd/system/aurora-demo.service
systemctl daemon-reload
systemctl enable --now aurora-demo
systemctl restart aurora-demo
for attempt in {1..20}; do
    if curl -fsS http://127.0.0.1:18082/api/health > "$backup/health.json"; then break; fi
    sleep 1
done
curl -fsS http://127.0.0.1:18082/api/health
install -m 644 "$release/deploy/aurora.nginx.conf" /etc/nginx/sites-available/aurora-demo
rm -f /etc/nginx/sites-enabled/csts
ln -sfn /etc/nginx/sites-available/aurora-demo /etc/nginx/sites-enabled/aurora-demo
if ! nginx -t; then
    rm -f /etc/nginx/sites-enabled/aurora-demo
    if [ -f "$backup/nginx.csts" ]; then ln -s /etc/nginx/sites-available/csts /etc/nginx/sites-enabled/csts; fi
    exit 1
fi
systemctl reload nginx
# Scoped removal: the label, not a broad name match, identifies the CSTS project.
mapfile -t containers < <(docker ps -aq --filter label=com.docker.compose.project=csts-docker)
if [ ${#containers[@]} -gt 0 ]; then docker stop -t 25 "${containers[@]}"; docker rm "${containers[@]}"; fi
mapfile -t networks < <(docker network ls -q --filter label=com.docker.compose.project=csts-docker)
if [ ${#networks[@]} -gt 0 ]; then docker network rm "${networks[@]}"; fi
for unit in csts-gateway csts-product csts-trade csts-user; do
    if [ -f "/etc/systemd/system/$unit.service" ]; then
        systemctl disable --now "$unit.service"
        rm -f "/etc/systemd/system/$unit.service"
    fi
done
systemctl daemon-reload
# Database volumes and old release files are retained as recovery material.
find /home/ubuntu/Project/resume -type f -print0 | sort -z | xargs -0 sha256sum > "$backup/resume.after.sha256"
diff -u "$backup/resume.before.sha256" "$backup/resume.after.sha256"
curl -fsS https://par73e.cn/resume/ > /dev/null
curl -fsS https://par73e.cn/api/health
printf '\nInstalled release %s; resume verified unchanged.\n' "$release_id"

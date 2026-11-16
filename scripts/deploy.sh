#!/usr/bin/env bash
set -euo pipefail

REPO=kericchhh/aux-battles
DIR="$HOME/aux-deploy"
STATE="$HOME/.local/state/aux-deploy"
HEALTH_URL=http://localhost:8000/api/healthz

log() { echo "$(date -u +%FT%TZ) $*" >&2; }

notify() {
  local token chat
  token=$(sed -n 's/^TELEGRAM_BOT_TOKEN=//p' "$DIR/.env" 2>/dev/null | tail -n1 | tr -d "\"'\r" || true)
  chat=$(sed -n 's/^TELEGRAM_CHAT_ID=//p' "$DIR/.env" 2>/dev/null | tail -n1 | tr -d "\"'\r" || true)
  log "$*"
  [ -n "$token" ] && [ -n "$chat" ] || return 0
  echo "url = \"https://api.telegram.org/bot$token/sendMessage\"" |
    curl -fsS --max-time 10 -o /dev/null -K - \
      --data-urlencode "chat_id=$chat" --data-urlencode "text=aux-battles: $*" ||
    log "telegram notification failed"
}

running() { [ -n "$(docker ps -q --filter "label=com.docker.compose.project=$1")" ]; }

compose() {
  (cd "$DIR" && AUX_VERSION="${1#v}" docker compose --progress quiet -p aux-prod --profile observability "${@:2}")
}

healthy() {
  for _ in 1 2 3 4 5 6; do
    curl -fsS -o /dev/null --max-time 5 "$HEALTH_URL" && return 0
    sleep 5
  done
  return 1
}

deploy() {
  git -C "$DIR" checkout --quiet --detach "refs/tags/$1" &&
    compose "$1" up -d --wait --wait-timeout 300 --remove-orphans &&
    healthy
}

mkdir -p "$STATE" "$DIR/backups"
exec 9>"$STATE/lock"
flock -n 9 || exit 0

if [ "${1:-}" = --if-running ] && ! running aux-prod; then exit 0; fi
if running aux-battles; then log "dev stack is running, skipping"; exit 0; fi

tag=$(curl -fsS --max-time 20 "https://api.github.com/repos/$REPO/releases/latest" | jq -r .tag_name)
[[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { log "unexpected tag '$tag'"; exit 1; }
current=$(cat "$STATE/current" 2>/dev/null || echo none)
[ "$tag" != "$current" ] || exit 0
if grep -qx "$tag" "$STATE/failed" 2>/dev/null; then exit 0; fi

git -C "$DIR" fetch --quiet origin "+refs/tags/$tag:refs/tags/$tag"
for app in api web worker; do docker pull --quiet "ghcr.io/kericchhh/aux-battles-$app:${tag#v}" >/dev/null; done

if [ "$current" != none ]; then
  compose "$current" stop web api worker
  if ! compose "$current" --profile backup run --rm -T backup; then
    echo "$tag" >> "$STATE/failed"
    compose "$current" start web api worker || true
    notify "backup failed, $tag not deployed"
    exit 1
  fi
fi

if deploy "$tag"; then
  echo "$tag" > "$STATE/current"
  notify "deployed $tag (was $current)"
  exit 0
fi

if [ "$current" != none ] && deploy "$current"; then
  echo "$tag" >> "$STATE/failed"
  notify "$tag failed health checks, rolled back to $current"
else
  echo "$tag" >> "$STATE/failed"
  notify "$tag failed and prod could not be started, check aux-prod"
fi
exit 1

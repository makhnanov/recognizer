#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="$PROJECT_DIR/recognizer"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

log "=== Recognizer stop script ==="
log "Binary path: $BINARY"

# --- Find and kill running process ---
log "Searching for running processes..."

pids=$(pgrep -f "$BINARY" 2>/dev/null || true)

if [ -z "$pids" ]; then
    log "No running processes found"
    exit 0
fi

log "Found processes:"
while IFS= read -r pid; do
    cmdline=$(cat /proc/"$pid"/cmdline 2>/dev/null | tr '\0' ' ' || echo "(unavailable)")
    log "  PID $pid: $cmdline"
done <<< "$pids"

log "Sending SIGTERM..."
while IFS= read -r pid; do
    if kill "$pid" 2>/dev/null; then
        log "  PID $pid: SIGTERM sent"
    else
        log "  PID $pid: already gone"
    fi
done <<< "$pids"

log "Waiting up to 5 seconds for graceful shutdown..."
for i in $(seq 1 10); do
    remaining=$(pgrep -f "$BINARY" 2>/dev/null || true)
    if [ -z "$remaining" ]; then
        log "All processes stopped (after ${i}×0.5s)"
        log "=== Done ==="
        exit 0
    fi
    sleep 0.5
done

remaining=$(pgrep -f "$BINARY" 2>/dev/null || true)
if [ -n "$remaining" ]; then
    log "Processes still alive, sending SIGKILL..."
    while IFS= read -r pid; do
        if kill -9 "$pid" 2>/dev/null; then
            log "  PID $pid: SIGKILL sent"
        fi
    done <<< "$remaining"
    sleep 0.5
fi

log "=== Done ==="

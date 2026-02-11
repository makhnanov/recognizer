#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="$PROJECT_DIR/recognizer"
PROCESS_NAME="recognizer"
LOG_FILE="$PROJECT_DIR/recognizer.log"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

log "=== Recognizer restart script ==="
log "Project dir: $PROJECT_DIR"
log "Binary path: $BINARY"
log "Log file:    $LOG_FILE"

# --- Find and kill running process ---
log "Searching for running '$PROCESS_NAME' processes..."

pids=$(pgrep -f "$BINARY" 2>/dev/null || true)

if [ -n "$pids" ]; then
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
            break
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
else
    log "No running processes found"
fi

# --- Build ---
log "Building recognizer..."
log "Running: go build -o $BINARY $PROJECT_DIR/recognizer.go"

cd "$PROJECT_DIR"
if go build -o "$BINARY" recognizer.go 2>&1; then
    log "Build successful"
    log "Binary size: $(stat -c%s "$BINARY") bytes"
else
    log "ERROR: Build failed, aborting"
    exit 1
fi

# --- Launch in background ---
log "Starting recognizer in background..."
log "Output redirected to: $LOG_FILE"

nohup "$BINARY" >> "$LOG_FILE" 2>&1 &
new_pid=$!
disown "$new_pid"

sleep 1

if kill -0 "$new_pid" 2>/dev/null; then
    log "Recognizer started successfully (PID $new_pid)"
else
    log "ERROR: Process failed to start, check $LOG_FILE"
    exit 1
fi

log "=== Done ==="

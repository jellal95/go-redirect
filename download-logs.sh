#!/bin/bash
# =============================================
# Fly.io Log Downloader & Cleaner (macOS/Linux)
# =============================================
# Requirements:
#   - flyctl installed & authenticated
#   - access to Fly.io app with /logs or /app/logs volume
# =============================================

set -euo pipefail

# === CONFIG & ARGS ===
LOCAL_LOGS_PATH="./logs"
NO_MERGE=false
NO_DELETE=false
DATE_FILTER=""
BACKUP=false

# Parse CLI args
while [[ $# -gt 0 ]]; do
  case $1 in
    -LocalLogsPath) LOCAL_LOGS_PATH="$2"; shift ;;
    -NoMerge) NO_MERGE=true ;;
    -NoDelete) NO_DELETE=true ;;
    -DateFilter) DATE_FILTER="$2"; shift ;;
    -Backup) BACKUP=true ;;
    *) echo "Unknown option: $1" ;;
  esac
  shift
done

echo "=== Fly.io Log Download & Cleanup ==="
echo ""
echo "Configuration:"
echo "  Local path: $LOCAL_LOGS_PATH"
echo "  Mode: $([[ $NO_MERGE == true ]] && echo 'Replace' || echo 'Merge')"
echo "  Delete remote: $([[ $NO_DELETE == true ]] && echo 'Keep remote files' || echo 'Delete after download')"
[[ -n $DATE_FILTER ]] && echo "  Date filter: $DATE_FILTER"
[[ $BACKUP == true ]] && echo "  Backup: Enabled"
echo ""

mkdir -p "$LOCAL_LOGS_PATH"

# === MACHINE INFO ===
echo "Getting Fly.io machine info..."
MACHINE_ID=$(fly machines list --json | jq -r '.[0].id')
if [[ -z "$MACHINE_ID" || "$MACHINE_ID" == "null" ]]; then
  echo "❌ No running Fly.io machines found. Exiting."
  exit 1
fi
echo "Found machine: $MACHINE_ID"

# === FIND REMOTE LOG FILES ===
echo "Listing log files on remote server..."
LOG_FILES=$(fly ssh console --machine "$MACHINE_ID" -C "find /logs -name '*.jsonl' -type f" 2>/dev/null || true)
if [[ -z "$LOG_FILES" || "$LOG_FILES" == *"No such file"* ]]; then
  echo "Trying /app/logs path..."
  LOG_FILES=$(fly ssh console --machine "$MACHINE_ID" -C "find /app/logs -name '*.jsonl' -type f" 2>/dev/null || true)
fi

if [[ -z "$LOG_FILES" || "$LOG_FILES" == *"No such file"* ]]; then
  echo "❌ No .jsonl log files found in /logs or /app/logs"
  exit 1
fi

# === FILTER FILES ===
LOG_FILES_FILTERED=()
while read -r file; do
  [[ -n "$file" && "$file" == /*.jsonl ]] && LOG_FILES_FILTERED+=("$file")
done <<< "$LOG_FILES"

if [[ -n "$DATE_FILTER" ]]; then
  LOG_FILES_FILTERED=($(printf "%s\n" "${LOG_FILES_FILTERED[@]}" | grep "$DATE_FILTER" || true))
fi

TOTAL=${#LOG_FILES_FILTERED[@]}
if [[ $TOTAL -eq 0 ]]; then
  echo "❌ No matching log files found."
  exit 1
fi

echo "Found $TOTAL log files."
echo ""

DOWNLOADED=0
DELETED=0
ERRORS=0

# === PROCESS FILES ===
for ((i=0; i<$TOTAL; i++)); do
  FILE="${LOG_FILES_FILTERED[$i]}"
  FILENAME=$(basename "$FILE")
  LOCAL_FILE="$LOCAL_LOGS_PATH/$FILENAME"
  TEMP_FILE="$LOCAL_FILE.temp"

  echo "[$((i+1))/$TOTAL] Processing $FILENAME"

  # Backup
  if [[ $BACKUP == true && -f "$LOCAL_FILE" ]]; then
    BACKUP_FILE="${LOCAL_FILE}.backup.$(date +%Y%m%d-%H%M%S)"
    cp "$LOCAL_FILE" "$BACKUP_FILE"
    echo "  → Backup created: $(basename "$BACKUP_FILE")"
  fi

  # Download (retry up to 3 times)
  SUCCESS=false
  for ATTEMPT in {1..3}; do
    fly ssh sftp get "$FILE" "$TEMP_FILE" --machine "$MACHINE_ID" >/dev/null 2>&1 && SUCCESS=true && break
    echo "  ⚠️ Retry $ATTEMPT/3..."
    sleep 2
  done

  if [[ $SUCCESS == true && -s "$TEMP_FILE" ]]; then
    echo "  ✅ Download successful ($(du -k "$TEMP_FILE" | cut -f1) KB)"
    if [[ -f "$LOCAL_FILE" && $NO_MERGE == false ]]; then
      # merge lines
      echo "  → Merging with existing file..."
      cat "$LOCAL_FILE" "$TEMP_FILE" | sort -u > "$LOCAL_FILE.merged"
      mv "$LOCAL_FILE.merged" "$LOCAL_FILE"
      rm -f "$TEMP_FILE"
    else
      mv -f "$TEMP_FILE" "$LOCAL_FILE"
      echo "  → Saved new file: $LOCAL_FILE"
    fi
    ((DOWNLOADED++))

    # Delete remote if allowed
    if [[ $NO_DELETE == false ]]; then
      fly ssh console --machine "$MACHINE_ID" -C "rm -f '$FILE'" >/dev/null 2>&1 || true
      echo "  🗑️ Deleted remote file"
      ((DELETED++))
    else
      echo "  ↩️ Keeping remote file"
    fi
  else
    echo "  ❌ Failed to download after retries"
    rm -f "$TEMP_FILE" || true
    ((ERRORS++))
  fi

  echo ""
done

# === SUMMARY ===
echo "============================================================"
echo " DOWNLOAD SUMMARY"
echo "============================================================"
echo " Files processed: $TOTAL"
echo " Downloaded:      $DOWNLOADED"
[[ $NO_DELETE == false ]] && echo " Deleted remote:  $DELETED" || echo " Remote kept"
[[ $ERRORS -gt 0 ]] && echo " Errors:          $ERRORS"
echo " Local logs path: $(realpath "$LOCAL_LOGS_PATH")"
echo "============================================================"

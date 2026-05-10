#!/usr/bin/env bash
# Extracts structured data from a Go service for doc generation.
# Usage: bash docs/reference/extractor.sh <service-dir>
# Output: docs/.docs-gen/<service-name>.extracted.json
# Run from monorepo root.

set -euo pipefail

SVC_DIR="${1:?Usage: $0 <service-dir>}"
SVC_NAME=$(basename "$SVC_DIR")
OUT_DIR="docs/.docs-gen"
OUT_FILE="$OUT_DIR/${SVC_NAME}.extracted.json"
QUEUENAME_FILE="bin-common-handler/models/outline/queuename.go"

mkdir -p "$OUT_DIR"

# Determine class from taxonomy
CLASS=$(grep -E "^\| $SVC_NAME \|" docs/reference/service-taxonomy.md | awk -F'|' '{print $3}' | xargs)

echo "Extracting $SVC_NAME (Class: $CLASS)" >&2

# Helper: run a pipeline stage and suppress non-zero exit (grep no-match = exit 1)
# Usage: safe_pipe <pipeline...>
# All intermediate greps must use || true to avoid pipefail cascade

# --- Routing table (listenhandler) ---
ROUTING="[]"
if [ -d "$SVC_DIR/pkg/listenhandler" ]; then
  ROUTING=$(
    { grep 'regexp\.MustCompile' "$SVC_DIR/pkg/listenhandler/main.go" 2>/dev/null || true; } \
    | { grep -oP '(?<=MustCompile\().*(?=\))' || true; } \
    | tr -d '"' | tr -d '`' \
    | sed 's/ + [a-zA-Z_][a-zA-Z_0-9]* + /{{UUID}}/g' \
    | sed 's/ + [a-zA-Z_][a-zA-Z_0-9]*$//g' \
    | sed 's/^[a-zA-Z_][a-zA-Z_0-9]* + //g' \
    | { grep -v '^$' || true; } \
    | jq -R '{pattern: .}' | jq -s '.'
  )
fi

# --- Events subscribed (cmd/*/main.go) ---
# Strategy: find the cmd/*/main.go that has subscribeTargets, extract QueueName constant names,
# then resolve them to string values via bin-common-handler/models/outline/queuename.go
EVENTS_SUB="[]"
MAIN_GO=$(grep -rl 'subscribeTargets' "$SVC_DIR/cmd/" 2>/dev/null | { grep 'main\.go' || true; } | head -1 || true)
if [ -n "$MAIN_GO" ]; then
  # Try resolving Go constants via queuename.go
  CONST_NAMES=$(
    { awk '/subscribeTargets.*:?=.*\[\]string\{/,/^\s*\}/' "$MAIN_GO" 2>/dev/null || true; } \
    | { grep -oP 'QueueName\w+' || true; }
  )
  if [ -n "$CONST_NAMES" ] && [ -f "$QUEUENAME_FILE" ]; then
    EVENTS_SUB=$(
      echo "$CONST_NAMES" \
      | while read -r const_name; do
          val=$(grep -E "^\s+${const_name}\s+" "$QUEUENAME_FILE" 2>/dev/null \
            | { grep -oP '"[^"]*"' || true; } | tr -d '"' || true)
          [ -n "$val" ] && echo "$val" || true
        done \
      | { grep -v '^$' || true; } \
      | jq -R '{queue_symbol: .}' | jq -s '.'
    )
  fi
  # Fallback: try raw string literals if no constants found
  if [ "$EVENTS_SUB" = "[]" ]; then
    EVENTS_SUB=$(
      { awk '/subscribeTargets.*:?=.*\[\]string\{/,/^\s*\}/' "$MAIN_GO" 2>/dev/null || true; } \
      | { grep -oP '"[^"]*"' || true; } | tr -d '"' \
      | { grep -v '^$' || true; } \
      | jq -R '{queue_symbol: .}' | jq -s '.'
    )
  fi
fi

# --- Events published (pkg/**/*.go + internal/**/*.go) ---
EVENTS_PUB="[]"
if [ -d "$SVC_DIR/pkg" ] || [ -d "$SVC_DIR/internal" ]; then
  PUB_DIRS=""
  [ -d "$SVC_DIR/pkg" ] && PUB_DIRS="${PUB_DIRS:+$PUB_DIRS }$SVC_DIR/pkg"
  [ -d "$SVC_DIR/internal" ] && PUB_DIRS="${PUB_DIRS:+$PUB_DIRS }$SVC_DIR/internal"
  EVENTS_PUB=$(
    { find $PUB_DIRS -name "*.go" \
      ! -name "*_test.go" ! -name "mock_*.go" 2>/dev/null \
      | xargs grep -hn 'PublishWebhookEvent\|CallPublishEvent' 2>/dev/null || true; } \
    | { grep -oP '\b(call|flow|agent|billing|ai|message|email|conference|queue|campaign|contact|conversation|customer|number|outdial|rag|registrar|route|sentinel|hook|storage|tag|talk|timeline|transcribe|transfer|tts|webhook)\.[A-Za-z]+\b' || true; } \
    | sort -u \
    | jq -R '{event_type_symbol: .}' | jq -s '.'
  )
fi

# --- Dependencies (go.mod replace directives) ---
DEPS="[]"
if [ -f "$SVC_DIR/go.mod" ]; then
  DEPS=$(
    { grep '^replace ' "$SVC_DIR/go.mod" 2>/dev/null || true; } \
    | sed 's/^replace \([^ ]*\) => \(.*\)/\1|\2/' \
    | { grep '|' || true; } \
    | jq -R 'split("|") | {module_path: .[0], local_path: .[1]}' \
    | jq -s '.'
  )
fi

# --- Config vars (internal/config/*.go takes priority over cmd/*/init.go) ---
CONFIG="[]"
CONFIG_SRC=$(ls "$SVC_DIR"/internal/config/*.go 2>/dev/null | head -1 \
  || ls "$SVC_DIR"/cmd/*/init.go 2>/dev/null | head -1 || true)
if [ -n "$CONFIG_SRC" ]; then
  CONFIG=$(
    { grep -n 'f\.String\|f\.Int\|f\.Bool\|f\.StringSlice' "$CONFIG_SRC" 2>/dev/null || true; } \
    | { grep -oP '"[a-z][a-z_0-9]+"' || true; } | tr -d '"' \
    | { grep -v '^$' || true; } \
    | jq -R '{flag: .}' | jq -s '.'
  )
fi

# --- Prometheus metrics (Name: "..." in opts blocks near prometheus.New*) ---
METRICS="[]"
METRIC_DIRS=""
[ -d "$SVC_DIR/pkg" ] && METRIC_DIRS="${METRIC_DIRS:+$METRIC_DIRS }$SVC_DIR/pkg"
[ -d "$SVC_DIR/internal" ] && METRIC_DIRS="${METRIC_DIRS:+$METRIC_DIRS }$SVC_DIR/internal"
if [ -n "$METRIC_DIRS" ]; then
  METRICS=$(
    { find $METRIC_DIRS -name "*.go" ! -name "*_test.go" 2>/dev/null \
      | xargs grep -hn 'Name:\s*"[a-z]' 2>/dev/null || true; } \
    | { grep -oP '"[a-z][a-z_0-9]+"' || true; } | tr -d '"' \
    | { grep -v '^$' || true; } \
    | sort -u \
    | jq -R '{name: .}' | jq -s '.'
  )
fi

# --- Missing fields ---
# Class exemptions:
#   A  = Standard Go RPC manager — requires routing_table, events_subscribed, config_vars
#   A2 = Event-driven worker, no inbound RPC — exempt from routing_table and events_subscribed
#   B  = HTTP/REST gateway — requires events_subscribed and config_vars (no routing per se)
#   C  = Shared library, no cmd/ — exempt from all
#   D  = Python/Alembic — exempt from all (no go.mod/config)
#   E  = OpenAPI codegen — exempt from all
MISSING="[]"
[ "$ROUTING" = "[]" ] && [ "$CLASS" = "A" ] && MISSING=$(echo "$MISSING" | jq '. + ["routing_table"]') || true
[ "$EVENTS_SUB" = "[]" ] && [ "$CLASS" = "A" ] && \
  MISSING=$(echo "$MISSING" | jq '. + ["events_subscribed"]') || true
[ "$CONFIG" = "[]" ] && [ "$CLASS" != "C" ] && [ "$CLASS" != "D" ] && [ "$CLASS" != "E" ] && \
  MISSING=$(echo "$MISSING" | jq '. + ["config_vars"]') || true

# --- Assemble JSON ---
jq -n \
  --arg schema_version "1.0" \
  --arg service_name "$SVC_NAME" \
  --arg service_class "$CLASS" \
  --argjson routing_table "$ROUTING" \
  --argjson events_subscribed "$EVENTS_SUB" \
  --argjson events_published "$EVENTS_PUB" \
  --argjson dependencies "$DEPS" \
  --argjson config_vars "$CONFIG" \
  --argjson metrics "$METRICS" \
  --argjson missing_fields "$MISSING" \
  '{
    schema_version: $schema_version,
    service_name: $service_name,
    service_class: $service_class,
    routing_table: $routing_table,
    events_subscribed: $events_subscribed,
    events_published: $events_published,
    dependencies: $dependencies,
    config_vars: $config_vars,
    metrics: $metrics,
    missing_fields: $missing_fields
  }' > "$OUT_FILE"

echo "Written: $OUT_FILE" >&2

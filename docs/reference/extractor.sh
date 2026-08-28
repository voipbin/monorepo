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

# --- Events subscribed ---
# Strategy (in priority order):
#   1. cmd/*/main.go with subscribeTargets as []string{} — standard multi-event pattern
#   2. cmd/*/main.go with subscribeTargets as string(...) — single-event pattern
#   3. pkg/subscribehandler/main.go package-level var subscribeTargets — timeline/api pattern
# QueueName constants are resolved via bin-common-handler/models/outline/queuename.go
EVENTS_SUB="[]"

# Source files to search (ordered by priority)
SUB_MAIN_GO=$(grep -rl 'subscribeTargets' "$SVC_DIR/cmd/" 2>/dev/null | { grep 'main\.go' || true; } | head -1 || true)
SUB_PKG_GO="$SVC_DIR/pkg/subscribehandler/main.go"

# Extract a `{ ... }` block starting at the first line matching open_regex,
# using brace-depth tracking instead of an awk range (`/open/,/close/`).
# A range pattern can't close correctly on an inline/empty literal such as
# `subscribeTargets := []string{}`: the opening pattern matches, but there is
# no later line consisting only of `}` to match the closing pattern, so the
# range never turns off and the scan spills into unrelated code (e.g. the
# enclosing function's body) until it happens to hit some other line that
# starts with `}`. Depth tracking closes on the same line when a literal
# opens and closes inline, and still spans multiple lines correctly for the
# non-empty case.
# LIMITATION: depth is counted from raw text (gsub over `{`/`}`), not a real
# parser, so an unbalanced brace inside a string literal or comment within
# the block could still mis-close. Acceptable for this best-effort doc
# extraction tool; the source patterns it targets (subscribeTargets slice
# literals) don't contain such braces in practice.
extract_brace_block() {
  local file="$1" open_regex="$2"
  # NOTE: the open regex is passed via the environment (ENVIRON), not `awk -v`.
  # `awk -v var=value` runs C-style backslash-escape processing on `value`,
  # which silently turns `\[`, `\]`, `\{` into `[`, `]`, `{` and produces an
  # invalid dynamic regex (this was tried and broke the multi-line case).
  # ENVIRON values are passed through verbatim.
  EXTRACT_OPEN_RE="$open_regex" awk '
    $0 ~ ENVIRON["EXTRACT_OPEN_RE"] { in_block = 1; depth = 0 }
    in_block {
      print
      depth += gsub(/\{/, "{")
      depth -= gsub(/\}/, "}")
      if (depth <= 0) in_block = 0
    }
  ' "$file" 2>/dev/null || true
}

# Resolve QueueName constants to their string values
resolve_const_names() {
  local const_names="$1"
  if [ -n "$const_names" ] && [ -f "$QUEUENAME_FILE" ]; then
    echo "$const_names" \
    | while read -r const_name; do
        val=$(grep -E "^\s+${const_name}\s+" "$QUEUENAME_FILE" 2>/dev/null \
          | { grep -oP '"[^"]*"' || true; } | tr -d '"' || true)
        [ -n "$val" ] && echo "$val" || true
      done \
    | { grep -v '^$' || true; }
  fi
}

if [ -n "$SUB_MAIN_GO" ]; then
  # Pattern 1: subscribeTargets = []string{...} or []string{} — multi-event list
  CONST_NAMES=$(
    extract_brace_block "$SUB_MAIN_GO" 'subscribeTargets.*:?=.*\[\]string\{' \
    | { grep -oP 'QueueName\w+' || true; }
  )
  if [ -n "$CONST_NAMES" ]; then
    resolved=$(resolve_const_names "$CONST_NAMES")
    if [ -n "$resolved" ]; then
      EVENTS_SUB=$(echo "$resolved" | jq -R '{queue_symbol: .}' | jq -s '.')
    fi
  fi
  # Fallback: try raw string literals from []string{} block
  if [ "$EVENTS_SUB" = "[]" ]; then
    EVENTS_SUB=$(
      extract_brace_block "$SUB_MAIN_GO" 'subscribeTargets.*:?=.*\[\]string\{' \
      | { grep -oP '"[^"]*"' || true; } | tr -d '"' \
      | { grep -v '^$' || true; } \
      | jq -R '{queue_symbol: .}' | jq -s '.'
    )
  fi

  # Pattern 2: subscribeTargets := string(commonoutline.QueueNameXxx) — single-event
  if [ "$EVENTS_SUB" = "[]" ]; then
    CONST_NAMES=$(
      { grep -oP 'subscribeTargets\s*:=\s*string\(commonoutline\.(QueueName\w+)\)' "$SUB_MAIN_GO" 2>/dev/null || true; } \
      | { grep -oP 'QueueName\w+' || true; }
    )
    if [ -n "$CONST_NAMES" ]; then
      resolved=$(resolve_const_names "$CONST_NAMES")
      if [ -n "$resolved" ]; then
        EVENTS_SUB=$(echo "$resolved" | jq -R '{queue_symbol: .}' | jq -s '.')
      fi
    fi
  fi
fi

# Pattern 3: package-level var subscribeTargets in pkg/subscribehandler/main.go
# Used by services like timeline-manager that define subscriptions inside the subscribehandler pkg
if [ "$EVENTS_SUB" = "[]" ] && [ -f "$SUB_PKG_GO" ]; then
  # Look for: var subscribeTargets = []commonoutline.QueueName{ ... } or []QueueName{ ... }
  CONST_NAMES=$(
    extract_brace_block "$SUB_PKG_GO" 'var[[:space:]]+subscribeTargets[[:space:]]*=[[:space:]]*\[\](commonoutline\.)?QueueName\{' \
    | { grep -oP 'QueueName\w+' || true; }
  )
  if [ -n "$CONST_NAMES" ]; then
    resolved=$(resolve_const_names "$CONST_NAMES")
    if [ -n "$resolved" ]; then
      EVENTS_SUB=$(echo "$resolved" | jq -R '{queue_symbol: .}' | jq -s '.')
    fi
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

# --- Config vars ---
# Searches internal/config/*.go (priority) or cmd/*/init.go
# Matches multiple flag registration patterns:
#   f.String(...)            — pflag.FlagSet shorthand (most common)
#   cmd.Flags().String(...)  — cobra direct (campaign-manager style)
#   pflag.String(...)        — top-level pflag (voip-* proxy style)
CONFIG="[]"
CONFIG_SRC=$(ls "$SVC_DIR"/internal/config/*.go 2>/dev/null | head -1 \
  || ls "$SVC_DIR"/cmd/*/init.go 2>/dev/null | head -1 || true)
if [ -n "$CONFIG_SRC" ]; then
  CONFIG=$(
    { grep -n \
        'f\.String\|f\.Int\|f\.Bool\|f\.StringSlice\|\.Flags()\.String\|\.Flags()\.Int\|\.Flags()\.Bool\|pflag\.String\|pflag\.Int\|pflag\.Bool' \
        "$CONFIG_SRC" 2>/dev/null || true; } \
    | { grep -oP '"[a-z][a-z_0-9]+"' || true; } | tr -d '"' \
    | { grep -v '^$' || true; } \
    | jq -R '{flag: .}' | jq -s '.'
  )
fi

# --- Prometheus metrics (Name: "..." in opts blocks near prometheus.New*) ---
# Scoped to the prometheus.New*/promauto.New* constructor call itself (via
# paren-depth tracking, same rationale and limitation as extract_brace_block
# above) so an unrelated struct literal that happens to have a "Name" field
# (e.g. a SIP provisioning XML entry list) doesn't get misattributed as a
# metric name.
METRICS="[]"
METRIC_DIRS=""
[ -d "$SVC_DIR/pkg" ] && METRIC_DIRS="${METRIC_DIRS:+$METRIC_DIRS }$SVC_DIR/pkg"
[ -d "$SVC_DIR/internal" ] && METRIC_DIRS="${METRIC_DIRS:+$METRIC_DIRS }$SVC_DIR/internal"
if [ -n "$METRIC_DIRS" ]; then
  METRICS=$(
    { find $METRIC_DIRS -name "*.go" ! -name "*_test.go" -print0 2>/dev/null \
      | xargs -0 -I{} awk '
          /(prometheus|promauto)\.New(Counter|Gauge|Histogram|Summary)(Vec)?\(/ { in_block = 1; depth = 0 }
          in_block {
            print
            depth += gsub(/\(/, "(")
            depth -= gsub(/\)/, ")")
            if (depth <= 0) in_block = 0
          }
        ' {} 2>/dev/null || true; } \
    | { grep -oP 'Name:\s*"\K[a-z][a-z_0-9]*(?=")' || true; } \
    | { grep -v '^$' || true; } \
    | sort -u \
    | jq -R '{name: .}' | jq -s '.'
  )
fi

# --- Missing fields ---
# Class exemptions:
#   A  = Standard Go RPC manager — requires routing_table and config_vars.
#         events_subscribed is only required if the service has a subscribehandler package
#         or explicit subscribeTargets wiring (some Class A services legitimately have none).
#   A2 = Event-driven worker, no inbound RPC — exempt from routing_table and events_subscribed
#   A+sub = Go RPC + embedded native daemon — exempt from events_subscribed
#   B  = HTTP/REST gateway — requires events_subscribed and config_vars (no routing per se)
#   C  = Shared library, no cmd/ — exempt from all
#   D  = Python/Alembic — exempt from all (no go.mod/config)
#   E  = OpenAPI codegen — exempt from all
MISSING="[]"
[ "$ROUTING" = "[]" ] && [ "$CLASS" = "A" ] && MISSING=$(echo "$MISSING" | jq '. + ["routing_table"]') || true

# events_subscribed: flag as missing only for Class A services that have subscribe wiring
# (a subscribehandler pkg OR subscribeTargets in cmd/) but extraction produced nothing.
if [ "$CLASS" = "A" ] && [ "$EVENTS_SUB" = "[]" ]; then
  HAS_SUBSCRIBE_WIRING="false"
  # Check for subscribehandler package
  [ -d "$SVC_DIR/pkg/subscribehandler" ] && HAS_SUBSCRIBE_WIRING="true"
  # Check for subscribeTargets in cmd/
  if [ "$HAS_SUBSCRIBE_WIRING" = "false" ]; then
    { grep -rl 'subscribeTargets\|subscribehandler\.New' "$SVC_DIR/cmd/" 2>/dev/null | grep -q '.' && HAS_SUBSCRIBE_WIRING="true"; } || true
  fi
  if [ "$HAS_SUBSCRIBE_WIRING" = "true" ]; then
    MISSING=$(echo "$MISSING" | jq '. + ["events_subscribed"]')
  fi
fi

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

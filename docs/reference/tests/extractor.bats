#!/usr/bin/env bats
# Regression + coverage tests for docs/reference/extractor.sh.
#
# extractor.sh had no test coverage at all before this file. It has been the
# source of two real, silently-wrong-output bugs found via manual
# investigation rather than an automated check:
#   - PR #1215: an awk range pattern (`/open/,/close/`) never closes on an
#     inline/empty Go slice literal (e.g. `subscribeTargets := []string{}`),
#     scanning past it into unrelated code. Fixed via brace/paren-depth
#     tracking (extract_brace_block, and the parallel METRICS section fix).
#   - VOIP-1411 (this PR): the Pattern-3 open-regex used gawk-only `\s`
#     whitespace shorthand, silently matching nothing under mawk.
# Both failure modes share a signature: the script still exits 0 and still
# prints "Written: ...json", but the JSON content is wrong. That's exactly
# the kind of bug a human skimming a diff or a "did it run" CI check won't
# catch - only asserting on the actual extracted values does. This suite
# covers the extraction patterns that share that risk (any brace/paren-depth
# scan, or any dynamic-regex-in-awk delivery), plus lighter smoke coverage
# of the remaining sections for completeness.
#
# Not covered here: EVENTS_PUB event-type-symbol extraction, and every
# combination of the class-exemption matrix in the "Missing fields" section
# - those don't share the depth-tracking/dynamic-regex risk this suite
# targets, and a couple of representative cases are enough to catch a
# structural regression in that logic.

load 'test_helper'

setup() {
    setup_extractor_test_env
}

teardown() {
    teardown_extractor_test_env
}

# --- Pattern 1: cmd/*/main.go, subscribeTargets := []string{...} / {} ------

@test "pattern 1: empty subscribeTargets literal produces an empty list, not garbage scraped from surrounding code (PR #1215 regression)" {
    add_service_class "svc-empty" "B"
    write_fixture_file "svc-empty/cmd/svc-empty/main.go" <<'EOF'
package main

func runSubscribe() {
	subscribeTargets := []string{}
	subHandler := subscribehandler.NewSubscribeHandler(
		sockHandler,
		reqHandler,
		queueNamePod,
		subscribeTargets,
		pubHandler,
	)

	if errRun := subHandler.Run(); errRun != nil {
		log.Errorf("Could not run the subscribe handler. err: %v", errRun)
		return
	}
}
EOF

    run run_extractor "svc-empty"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-empty" '.events_subscribed')"
    [ "$result" = "[]" ]
}

@test "pattern 1: empty subscribeTargets literal is also correct under mawk (ENVIRON-based regex delivery, not awk -v)" {
    use_mawk
    add_service_class "svc-empty-mawk" "B"
    write_fixture_file "svc-empty-mawk/cmd/svc-empty-mawk/main.go" <<'EOF'
package main

func runSubscribe() {
	subscribeTargets := []string{}
	_ = subscribeTargets
}
EOF

    run run_extractor "svc-empty-mawk"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-empty-mawk" '.events_subscribed')"
    [ "$result" = "[]" ]
}

@test "pattern 1: multi-line QueueName const list resolves through queuename.go" {
    add_service_class "svc-multi" "A"
    add_queue_name "QueueNameCallEvent" "bin-manager.call-manager.event"
    add_queue_name "QueueNameCustomerEvent" "bin-manager.customer-manager.event"
    write_fixture_file "svc-multi/cmd/svc-multi/main.go" <<'EOF'
package main

func runSubscribe() {
	subscribeTargets := []string{
		string(commonoutline.QueueNameCallEvent),
		string(commonoutline.QueueNameCustomerEvent),
	}
	_ = subscribeTargets
}
EOF

    run run_extractor "svc-multi"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-multi" '[.events_subscribed[].queue_symbol] | sort')"
    [ "$result" = '["bin-manager.call-manager.event","bin-manager.customer-manager.event"]' ]
}

@test "pattern 1: raw string literal fallback works when no QueueName consts are used" {
    add_service_class "svc-raw" "A"
    write_fixture_file "svc-raw/cmd/svc-raw/main.go" <<'EOF'
package main

func runSubscribe() {
	subscribeTargets := []string{
		"raw.queue.one",
		"raw.queue.two",
	}
	_ = subscribeTargets
}
EOF

    run run_extractor "svc-raw"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-raw" '[.events_subscribed[].queue_symbol] | sort')"
    [ "$result" = '["raw.queue.one","raw.queue.two"]' ]
}

# --- Pattern 2: cmd/*/main.go, subscribeTargets := string(...) -------------

@test "pattern 2: single-event string(commonoutline.QueueNameXxx) resolves correctly" {
    add_service_class "svc-single" "A"
    add_queue_name "QueueNameFlowEvent" "bin-manager.flow-manager.event"
    write_fixture_file "svc-single/cmd/svc-single/main.go" <<'EOF'
package main

func runSubscribe() {
	subscribeTargets := string(commonoutline.QueueNameFlowEvent)
	_ = subscribeTargets
}
EOF

    run run_extractor "svc-single"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-single" '.events_subscribed')"
    [ "$result" = '[{"queue_symbol":"bin-manager.flow-manager.event"}]' ]
}

# --- Pattern 3: pkg/subscribehandler/main.go, var subscribeTargets = ... ---

@test "pattern 3: empty var subscribeTargets literal produces an empty list" {
    add_service_class "svc-pkg-empty" "A"
    write_fixture_file "svc-pkg-empty/pkg/subscribehandler/main.go" <<'EOF'
package subscribehandler

var subscribeTargets = []commonoutline.QueueName{}

func Run() error {
	return nil
}
EOF

    run run_extractor "svc-pkg-empty"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-pkg-empty" '.events_subscribed')"
    [ "$result" = "[]" ]
}

@test "pattern 3: multi-line package-level var subscribeTargets resolves through queuename.go" {
    add_service_class "svc-pkg-multi" "A"
    add_queue_name "QueueNameAIEvent" "bin-manager.ai-manager.event"
    add_queue_name "QueueNameAgentEvent" "bin-manager.agent-manager.event"
    write_fixture_file "svc-pkg-multi/pkg/subscribehandler/main.go" <<'EOF'
package subscribehandler

var subscribeTargets = []commonoutline.QueueName{
	commonoutline.QueueNameAIEvent,
	commonoutline.QueueNameAgentEvent,
}

func Run() error {
	return nil
}
EOF

    run run_extractor "svc-pkg-multi"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-pkg-multi" '[.events_subscribed[].queue_symbol] | sort')"
    [ "$result" = '["bin-manager.agent-manager.event","bin-manager.ai-manager.event"]' ]
}

@test "pattern 3: matches under mawk too (VOIP-1411 regression - gawk-only \\s previously matched nothing under mawk)" {
    use_mawk
    add_service_class "svc-pkg-mawk" "A"
    add_queue_name "QueueNameCallEvent" "bin-manager.call-manager.event"
    write_fixture_file "svc-pkg-mawk/pkg/subscribehandler/main.go" <<'EOF'
package subscribehandler

var subscribeTargets = []commonoutline.QueueName{
	commonoutline.QueueNameCallEvent,
}

func Run() error {
	return nil
}
EOF

    run run_extractor "svc-pkg-mawk"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-pkg-mawk" '.events_subscribed')"
    [ "$result" = '[{"queue_symbol":"bin-manager.call-manager.event"}]' ]
}

# --- Prometheus metrics: scoped to prometheus.New*/promauto.New* calls -----

@test "metrics: Name inside a prometheus.NewCounterVec(...) call is captured" {
    add_service_class "svc-metrics" "A"
    write_fixture_file "svc-metrics/pkg/somehandler/metrics.go" <<'EOF'
package somehandler

var (
	promCallCreateTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "call_create_total",
			Help: "Total number of calls created.",
		},
		[]string{"status"},
	)
)
EOF

    run run_extractor "svc-metrics"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-metrics" '.metrics')"
    [ "$result" = '[{"name":"call_create_total"}]' ]
}

@test "metrics: a Name field on an unrelated struct (not inside prometheus.New*) is NOT captured (false-positive regression)" {
    add_service_class "svc-metrics-fp" "A"
    write_fixture_file "svc-metrics-fp/pkg/somehandler/provisioning.go" <<'EOF'
package somehandler

type entry struct {
	Name  string
	Value string
}

var entries = []entry{
	{Name: "domain", Value: "example.com"},
	{Name: "username", Value: "1001"},
}
EOF

    run run_extractor "svc-metrics-fp"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-metrics-fp" '.metrics')"
    [ "$result" = "[]" ]
}

@test "metrics: a real metric and an unrelated Name-bearing struct in the same file only extracts the real metric (direct repro of the bin-api-manager bug)" {
    add_service_class "svc-metrics-mixed" "A"
    write_fixture_file "svc-metrics-mixed/pkg/somehandler/main.go" <<'EOF'
package somehandler

var (
	promAuthDelegateTokenIssuedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_delegate_token_issued_total",
		Help: "Total number of delegate tokens issued.",
	})
)

type provisioningEntry struct {
	Name  string
	Value string
}

var provisioningEntries = []provisioningEntry{
	{Name: "passwd", Value: "secret"},
	{Name: "realm", Value: "example.com"},
}
EOF

    run run_extractor "svc-metrics-mixed"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-metrics-mixed" '.metrics')"
    [ "$result" = '[{"name":"auth_delegate_token_issued_total"}]' ]
}

@test "metrics: multiple prometheus.New* calls across multiple files are all captured, deduped, and sorted" {
    add_service_class "svc-metrics-multi" "A"
    write_fixture_file "svc-metrics-multi/pkg/handlera/main.go" <<'EOF'
package handlera

var promB = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "b_total",
})
EOF
    write_fixture_file "svc-metrics-multi/pkg/handlerb/main.go" <<'EOF'
package handlerb

var promA = prometheus.NewHistogram(prometheus.HistogramOpts{
	Name: "a_duration_seconds",
})
EOF

    run run_extractor "svc-metrics-multi"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-metrics-multi" '[.metrics[].name]')"
    [ "$result" = '["a_duration_seconds","b_total"]' ]
}

# --- Lighter smoke coverage: routing / config / dependencies ---------------

@test "routing: extracts regexp.MustCompile patterns from listenhandler, collapsing a + var + interpolation into {{UUID}}" {
    add_service_class "svc-routing" "A"
    write_fixture_file "svc-routing/pkg/listenhandler/main.go" <<'EOF'
package listenhandler

var regV1CallsGet = regexp.MustCompile("/v1/calls$")
var regV1CallsIDGet = regexp.MustCompile("/v1/calls/" + id + "$")
EOF

    run run_extractor "svc-routing"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-routing" '[.routing_table[].pattern] | sort')"
    [ "$result" = '["/v1/calls$","/v1/calls/{{UUID}}$"]' ]
}

@test "config: extracts pflag flag registrations from internal/config" {
    add_service_class "svc-config" "A"
    write_fixture_file "svc-config/internal/config/main.go" <<'EOF'
package config

func Init(f *pflag.FlagSet) {
	f.String("database_dsn", "", "database DSN")
	f.Int("redis_database", 0, "redis database index")
}
EOF

    run run_extractor "svc-config"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-config" '[.config_vars[].flag] | sort')"
    [ "$result" = '["database_dsn","redis_database"]' ]
}

@test "dependencies: extracts go.mod replace directives" {
    add_service_class "svc-deps" "A"
    write_fixture_file "svc-deps/go.mod" <<'EOF'
module monorepo/svc-deps

go 1.22

require monorepo/bin-common-handler v0.0.0

replace monorepo/bin-common-handler => ../bin-common-handler
EOF

    run run_extractor "svc-deps"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-deps" '.dependencies')"
    [ "$result" = '[{"module_path":"monorepo/bin-common-handler","local_path":"../bin-common-handler"}]' ]
}

# --- Class lookup and missing_fields exemptions -----------------------------

@test "class lookup: reads the service's class from the taxonomy table" {
    add_service_class "svc-class-lookup" "A2" "event-driven worker"

    run run_extractor "svc-class-lookup"
    [ "$status" -eq 0 ]
    [[ "$output" == *"Class: A2"* ]]

    result="$(extracted_json_field "svc-class-lookup" '.service_class')"
    [ "$result" = '"A2"' ]
}

@test "missing_fields: Class A service with no routing table is flagged missing, and only that" {
    add_service_class "svc-missing-routing" "A"
    write_fixture_file "svc-missing-routing/go.mod" <<'EOF'
module monorepo/svc-missing-routing
go 1.22
EOF
    # Give the fixture a populated config_vars source so config_vars isn't
    # ALSO flagged missing - otherwise a substring check on "routing_table"
    # would still pass even if the routing_table exemption logic were
    # broken and flagging everything, pinning nothing.
    write_fixture_file "svc-missing-routing/internal/config/main.go" <<'EOF'
package config

func Init(f *pflag.FlagSet) {
	f.String("database_dsn", "", "database DSN")
}
EOF

    run run_extractor "svc-missing-routing"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-missing-routing" '.missing_fields')"
    [ "$result" = '["routing_table"]' ]
}

@test "missing_fields: Class C (shared library) service is exempt from all missing-field checks" {
    add_service_class "svc-shared-lib" "C"

    run run_extractor "svc-shared-lib"
    [ "$status" -eq 0 ]

    result="$(extracted_json_field "svc-shared-lib" '.missing_fields')"
    [ "$result" = "[]" ]
}

# --- Usage -------------------------------------------------------------------

@test "usage: exits non-zero with no service-dir argument" {
    run bash "$EXTRACTOR_SH"
    [ "$status" -ne 0 ]
    [[ "$output" == *"Usage:"* ]]
}

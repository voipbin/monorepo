package config

import (
	"reflect"
	"strings"
	"testing"
)

// Test_Validate_listenSetters drives each listen-related setter and observes
// the result through Validate, the consumer of those fields.
func Test_Validate_listenSetters(t *testing.T) {
	tests := []struct {
		name string

		apply func()

		expectErr string // empty means Validate must pass
	}{
		{name: "poll interval positive", apply: func() { SetAIcallListenConfbridgeReadyPollIntervalForTest(5) }},
		{name: "poll interval zero", apply: func() { SetAIcallListenConfbridgeReadyPollIntervalForTest(0) }, expectErr: "aicall_listen_confbridge_ready_poll_interval_seconds must be > 0, got 0"},
		{name: "max wait below goroutine timeout", apply: func() { SetAIcallListenConfbridgeReadyMaxWaitForTest(44) }},
		{name: "max wait equals goroutine timeout", apply: func() { SetAIcallListenConfbridgeReadyMaxWaitForTest(45) }, expectErr: "(45) < aicall_listen_ensure_goroutine_timeout_seconds (45)"},
		{name: "max wait negative", apply: func() { SetAIcallListenConfbridgeReadyMaxWaitForTest(-1) }, expectErr: "aicall_listen_confbridge_ready_max_wait_seconds must be > 0, got -1"},
		{name: "start lock ttl above goroutine timeout", apply: func() { SetAIcallListenStartLockTTLForTest(46) }},
		{name: "start lock ttl equals goroutine timeout", apply: func() { SetAIcallListenStartLockTTLForTest(45) }, expectErr: "aicall_listen_start_lock_ttl_seconds (45) must hold"},
		{name: "start lock ttl zero", apply: func() { SetAIcallListenStartLockTTLForTest(0) }, expectErr: "aicall_listen_start_lock_ttl_seconds must be > 0, got 0"},
		{name: "max turns one", apply: func() { SetAIcallListenMaxTurnsPerAIcallForTest(1) }},
		{name: "max turns zero", apply: func() { SetAIcallListenMaxTurnsPerAIcallForTest(0) }, expectErr: "aicall_listen_max_turns_per_aicall must be > 0, got 0"},
		{name: "window size one", apply: func() { SetAIcallListenWindowSizeForTest(1) }},
		{name: "window size negative", apply: func() { SetAIcallListenWindowSizeForTest(-3) }, expectErr: "aicall_listen_window_size must be > 0, got -3"},
		{name: "qa context size one", apply: func() { SetAIcallListenQAContextSizeForTest(1) }},
		{name: "qa context size zero", apply: func() { SetAIcallListenQAContextSizeForTest(0) }, expectErr: "aicall_listen_qa_context_size must be > 0, got 0"},
		{name: "max message chars one", apply: func() { SetAIcallListenConversationMaxMessageCharsForTest(1) }},
		{name: "max message chars zero", apply: func() { SetAIcallListenConversationMaxMessageCharsForTest(0) }, expectErr: "aicall_listen_conversation_max_message_chars must be > 0, got 0"},
		{name: "flush jitter zero is allowed", apply: func() { SetAIcallListenConversationFlushJitterMsForTest(0) }},
		{name: "flush jitter negative", apply: func() { SetAIcallListenConversationFlushJitterMsForTest(-1) }, expectErr: "aicall_listen_conversation_flush_jitter_ms must be >= 0, got -1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidConfigForTest(t)

			if err := Validate(); err != nil {
				t.Fatalf("baseline config must be valid, got: %v", err)
			}

			tt.apply()

			err := Validate()
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("expected valid config, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.expectErr)
			}
			if !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("error = %q, expect to contain %q", err.Error(), tt.expectErr)
			}
		})
	}
}

// Test_setters_touchOnlyTheirField checks that the two plain setters change
// exactly one field of the shared config.
func Test_setters_touchOnlyTheirField(t *testing.T) {
	tests := []struct {
		name string

		apply  func()
		expect func(c *Config)
	}{
		{
			name:   "send cooldown seconds",
			apply:  func() { SetAIcallSendCooldownSecondsForTest(17) },
			expect: func(c *Config) { c.AIcallSendCooldownSeconds = 17 },
		},
		{
			name:   "contact case recreate rate limit minutes",
			apply:  func() { SetAIcallContactCaseRecreateRateLimitMinutesForTest(23) },
			expect: func(c *Config) { c.AIcallContactCaseRecreateRateLimitMinutes = 23 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreGlobalConfig(t)
			globalConfig = Config{}

			expect := Config{}
			tt.expect(&expect)

			tt.apply()

			if !reflect.DeepEqual(*Get(), expect) {
				t.Errorf("config = %+v, expect: %+v", *Get(), expect)
			}
		})
	}
}

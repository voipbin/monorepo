package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func Test_resolveUUID(t *testing.T) {
	tests := []struct {
		name  string
		value string

		expectErr    string
		expectResult string
	}{
		{
			name:         "valid uuid",
			value:        "a1b2c3d4-0000-4000-8000-000000000001",
			expectResult: "a1b2c3d4-0000-4000-8000-000000000001",
		},
		{
			name:      "empty value is required",
			value:     "",
			expectErr: "Call ID is required",
		},
		{
			name:      "malformed value is rejected with the input in the message",
			value:     "not-a-uuid",
			expectErr: "invalid format for Call ID: 'not-a-uuid' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("call-id", tt.value)

			res, err := resolveUUID("call-id", "Call ID")
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr {
					t.Fatalf("Wrong error. expected: %q, got: %v", tt.expectErr, err)
				}
				if res.String() != "00000000-0000-0000-0000-000000000000" {
					t.Errorf("a nil uuid must be returned on error, got: %s", res)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.String() != tt.expectResult {
				t.Errorf("Wrong uuid. expected: %s, got: %s", tt.expectResult, res)
			}
		})
	}
}

func Test_printJSON(t *testing.T) {
	t.Run("indents with two spaces", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("could not create a pipe: %v", err)
		}
		os.Stdout = w
		t.Cleanup(func() { os.Stdout = oldStdout })

		errPrint := printJSON(map[string]any{"a": 1, "b": []string{"x"}})
		_ = w.Close()
		os.Stdout = oldStdout

		out, _ := io.ReadAll(r)
		if errPrint != nil {
			t.Fatalf("unexpected error: %v", errPrint)
		}

		expect := "{\n  \"a\": 1,\n  \"b\": [\n    \"x\"\n  ]\n}\n"
		if string(out) != expect {
			t.Errorf("Wrong output.\nexpected: %q\ngot:      %q", expect, string(out))
		}
	})

	t.Run("values that cannot be encoded are reported", func(t *testing.T) {
		err := printJSON(make(chan int))
		if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
			t.Fatalf("expected a marshal error, got: %v", err)
		}
	})
}

func Test_runCreate_inputValidation(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string

		expectErr string
	}{
		{
			name:      "missing call id",
			values:    map[string]string{"text": "hello"},
			expectErr: "failed to resolve call ID",
		},
		{
			name:      "malformed call id",
			values:    map[string]string{"call-id": "xyz", "text": "hello"},
			expectErr: "failed to resolve call ID",
		},
		{
			name:      "missing text",
			values:    map[string]string{"call-id": "a1b2c3d4-0000-4000-8000-000000000001"},
			expectErr: "text is required",
		},
		{
			name: "unknown provider",
			values: map[string]string{
				"call-id":  "a1b2c3d4-0000-4000-8000-000000000001",
				"text":     "hello",
				"provider": "other",
			},
			expectErr: "invalid provider: other (must be gcp or aws)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			for k, v := range tt.values {
				viper.Set(k, v)
			}

			err := runCreate(cmdCreate(), nil)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Fatalf("Wrong error. expected to contain %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

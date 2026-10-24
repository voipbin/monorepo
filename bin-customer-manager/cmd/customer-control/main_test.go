package main

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// setViperKeys sets the given keys as overrides and restores them when the
// test ends. viper has no API to delete a key, so a nil override is used to
// release a key that was not set before.
func setViperKeys(t *testing.T, keys map[string]any) {
	t.Helper()

	for key, val := range keys {
		wasSet := viper.IsSet(key)
		old := viper.Get(key)
		viper.Set(key, val)

		t.Cleanup(func() {
			if wasSet {
				viper.Set(key, old)
				return
			}
			viper.Set(key, nil)
		})
	}
}

func Test_resolveUUID(t *testing.T) {
	validID := "9d8e7f60-0000-4000-8000-000000000001"

	tests := []struct {
		name      string
		value     string
		expectID  uuid.UUID
		expectErr string
	}{
		{
			name:      "empty",
			value:     "",
			expectID:  uuid.Nil,
			expectErr: "Customer ID is required",
		},
		{
			name:      "not a uuid",
			value:     "not-a-uuid",
			expectID:  uuid.Nil,
			expectErr: "invalid format for Customer ID: 'not-a-uuid' is not a valid UUID",
		},
		{
			name:     "valid",
			value:    validID,
			expectID: uuid.FromStringOrNil(validID),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setViperKeys(t, map[string]any{"resolve-uuid-key": tt.value})

			res, err := resolveUUID("resolve-uuid-key", "Customer ID")
			if res != tt.expectID {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectID, res)
			}
			if tt.expectErr == "" {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_resolveString(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		expectErr string
	}{
		{name: "empty", value: "", expectErr: "Email is required"},
		{name: "value", value: "customer-a@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setViperKeys(t, map[string]any{"resolve-string-key": tt.value})

			res, err := resolveString("resolve-string-key", "Email")
			if tt.expectErr != "" {
				if res != "" || err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect: %q, got: %q, %v", tt.expectErr, res, err)
				}
				return
			}
			if err != nil || res != tt.value {
				t.Errorf("Wrong match. expect: %q, got: %q, %v", tt.value, res, err)
			}
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe. err: %v", err)
	}

	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("could not close the pipe. err: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("could not read the pipe. err: %v", err)
	}
	return string(out)
}

func Test_printJSON(t *testing.T) {
	var err error
	out := captureStdout(t, func() {
		err = printJSON(map[string]any{"name": "customer A", "count": 2})
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := "{\n  \"count\": 2,\n  \"name\": \"customer A\"\n}\n"
	if out != expect {
		t.Errorf("Wrong match. expect: %q, got: %q", expect, out)
	}
}

func Test_printJSON_marshalError(t *testing.T) {
	var err error
	out := captureStdout(t, func() {
		err = printJSON(make(chan int))
	})
	if err == nil || !strings.Contains(err.Error(), "failed to marshal JSON") {
		t.Errorf("Wrong match. expect: marshal error, got: %v", err)
	}
	if out != "" {
		t.Errorf("Wrong match. expect: no output, got: %q", out)
	}
}

// Test_run_validationErrors covers the argument checks that run before any
// connection is opened. The expected messages and viper keys are written out
// here independently of the command source.
func Test_run_validationErrors(t *testing.T) {
	validID := "9d8e7f60-0000-4000-8000-0000000000a1"
	validBilling := "9d8e7f60-0000-4000-8000-0000000000a2"

	tests := []struct {
		name      string
		run       func(cmd *cobra.Command, args []string) error
		keys      map[string]any
		expectErr string
	}{
		{
			name:      "create without email",
			run:       runCreate,
			keys:      map[string]any{"email": ""},
			expectErr: "invalid email: Email is required",
		},
		{
			name:      "update with a bad id",
			run:       runUpdate,
			keys:      map[string]any{"id": "bad", "email": "customer-a@example.com"},
			expectErr: "invalid customer ID: invalid format for Customer ID: 'bad' is not a valid UUID",
		},
		{
			name:      "update without email",
			run:       runUpdate,
			keys:      map[string]any{"id": validID, "email": ""},
			expectErr: "invalid email: Email is required",
		},
		{
			name:      "update billing account with a bad id",
			run:       runUpdateBillingAccount,
			keys:      map[string]any{"id": "", "billing-account-id": validBilling},
			expectErr: "invalid customer ID: Customer ID is required",
		},
		{
			name:      "update billing account with a bad billing account id",
			run:       runUpdateBillingAccount,
			keys:      map[string]any{"id": validID, "billing-account-id": "bad"},
			expectErr: "invalid billing account ID: invalid format for Billing Account ID: 'bad' is not a valid UUID",
		},
		{
			name:      "get with a bad id",
			run:       runGet,
			keys:      map[string]any{"id": "bad"},
			expectErr: "invalid customer ID: invalid format for Customer ID: 'bad' is not a valid UUID",
		},
		{
			name:      "delete with a bad id",
			run:       runDelete,
			keys:      map[string]any{"id": ""},
			expectErr: "invalid customer ID: Customer ID is required",
		},
		{
			name:      "freeze with a bad id",
			run:       runFreeze,
			keys:      map[string]any{"id": "bad"},
			expectErr: "invalid customer ID: invalid format for Customer ID: 'bad' is not a valid UUID",
		},
		{
			name:      "recover with a bad id",
			run:       runRecover,
			keys:      map[string]any{"id": ""},
			expectErr: "invalid customer ID: Customer ID is required",
		},
		{
			name:      "set identity verification with a bad id",
			run:       runSetIdentityVerification,
			keys:      map[string]any{"id": "bad", "status": "verified"},
			expectErr: "invalid customer ID: invalid format for Customer ID: 'bad' is not a valid UUID",
		},
		{
			name:      "set identity verification without status",
			run:       runSetIdentityVerification,
			keys:      map[string]any{"id": validID, "status": ""},
			expectErr: "invalid status: Status is required",
		},
		{
			name:      "set identity verification with an undefined status",
			run:       runSetIdentityVerification,
			keys:      map[string]any{"id": validID, "status": "approved"},
			expectErr: "invalid status 'approved': must be one of none, pending, verified, rejected",
		},
		{
			name:      "accesskey create with a bad customer id",
			run:       runAccesskeyCreate,
			keys:      map[string]any{"customer-id": "bad", "name": "key", "expire": ""},
			expectErr: "invalid customer ID: invalid format for Customer ID: 'bad' is not a valid UUID",
		},
		{
			name:      "accesskey create without name",
			run:       runAccesskeyCreate,
			keys:      map[string]any{"customer-id": validID, "name": "", "expire": ""},
			expectErr: "invalid name: Name is required",
		},
		{
			name:      "accesskey create with a bad expire",
			run:       runAccesskeyCreate,
			keys:      map[string]any{"customer-id": validID, "name": "key", "expire": "tomorrow"},
			expectErr: "invalid expire duration format: time: invalid duration \"tomorrow\"",
		},
		{
			name:      "accesskey get with a bad id",
			run:       runAccesskeyGet,
			keys:      map[string]any{"id": "bad"},
			expectErr: "invalid accesskey ID: invalid format for Accesskey ID: 'bad' is not a valid UUID",
		},
		{
			name:      "accesskey update with a bad id",
			run:       runAccesskeyUpdate,
			keys:      map[string]any{"id": ""},
			expectErr: "invalid accesskey ID: Accesskey ID is required",
		},
		{
			name:      "accesskey delete with a bad id",
			run:       runAccesskeyDelete,
			keys:      map[string]any{"id": "bad"},
			expectErr: "invalid accesskey ID: invalid format for Accesskey ID: 'bad' is not a valid UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setViperKeys(t, tt.keys)

			err := tt.run(&cobra.Command{}, nil)
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong match. expect: %q, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_initCommand_tree(t *testing.T) {
	// initCommand applies the process-wide log settings; restore them
	oldLevel := logrus.GetLevel()
	oldFormatter := logrus.StandardLogger().Formatter
	t.Cleanup(func() {
		logrus.SetLevel(oldLevel)
		logrus.SetFormatter(oldFormatter)
	})

	root := initCommand()

	// the expected commands and the flags each one reads, written out
	// independently of the command source
	tests := []struct {
		path        []string
		flags       []string
		expectValue map[string]string
	}{
		{[]string{"customer", "create"}, []string{"name", "detail", "email", "phone-number", "address", "webhook-method", "webhook-uri"}, map[string]string{"webhook-method": "POST"}},
		{[]string{"customer", "get"}, []string{"id"}, nil},
		{[]string{"customer", "list"}, []string{"limit", "token"}, map[string]string{"limit": "100"}},
		{[]string{"customer", "update"}, []string{"id", "name", "detail", "email", "phone-number", "address", "webhook-method", "webhook-uri"}, map[string]string{"webhook-method": "POST"}},
		{[]string{"customer", "update-billing-account"}, []string{"id", "billing-account-id"}, nil},
		{[]string{"customer", "delete"}, []string{"id"}, nil},
		{[]string{"customer", "freeze"}, []string{"id"}, nil},
		{[]string{"customer", "recover"}, []string{"id"}, nil},
		{[]string{"customer", "set-identity-verification"}, []string{"id", "status"}, nil},
		{[]string{"accesskey", "create"}, []string{"customer-id", "name", "detail", "expire"}, nil},
		{[]string{"accesskey", "get"}, []string{"id"}, nil},
		{[]string{"accesskey", "list"}, []string{"customer-id", "size", "token"}, map[string]string{"size": "10"}},
		{[]string{"accesskey", "update"}, []string{"id", "name", "detail"}, nil},
		{[]string{"accesskey", "delete"}, []string{"id"}, nil},
		{[]string{"migrate"}, nil, nil},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.path, " "), func(t *testing.T) {
			cmd, _, err := root.Find(tt.path)
			if err != nil || cmd == nil || cmd.Name() != tt.path[len(tt.path)-1] {
				t.Fatalf("Wrong match. expect the command to be registered, got: %v, %v", cmd, err)
			}

			for _, name := range tt.flags {
				if cmd.Flags().Lookup(name) == nil {
					t.Errorf("Wrong match. expect the flag %q to be defined", name)
				}
			}
			for name, value := range tt.expectValue {
				flag := cmd.Flags().Lookup(name)
				if flag == nil || flag.DefValue != value {
					t.Errorf("Wrong match. expect the default of %q to be %q, got: %v", name, value, flag)
				}
			}
		})
	}

	// the config flags are defined once on the root
	if root.PersistentFlags().Lookup("database_dsn") == nil {
		t.Errorf("Wrong match. expect the config flags to be bootstrapped on the root")
	}
}

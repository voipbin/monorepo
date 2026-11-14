package providerhandler

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
)

const (
	fakeEnvTelnyxKey = "fake-env-telnyx-key"
	fakeEnvTwilioKey = "fake-env-twilio-key"
)

// setCredentialEnv sets the five credential variables of the test, an empty value is left unset.
func setCredentialEnv(t *testing.T, values map[string]string) {
	t.Helper()

	for _, name := range []string{envTelnyxAPIKey, envTelnyxConnectionID, envTelnyxProfileID, envTwilioAccountSID, envTwilioAuthToken} {
		t.Setenv(name, values[name])
	}
}

// garbageRow returns a row whose stored ciphertext cannot be decrypted by any key.
func garbageRow(name number.ProviderName, enabled bool, config map[string]string) *provider.Provider {
	return &provider.Provider{
		ProviderName:     name,
		Enabled:          enabled,
		Priority:         5,
		Config:           config,
		SecretCiphertext: []byte("not-a-ciphertext"),
		KeyVersion:       9,
	}
}

func Test_resolve_env(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		providerName number.ProviderName
		env          map[string]string
		row          *provider.Provider

		expectSource    provider.CredentialSource
		expectHasSecret bool
		expectUsable    bool
		expectSecret    string
		expectConfig    map[string]string
	}{
		{
			name:         "telnyx without a row: the variables are enough",
			providerName: number.ProviderNameTelnyx,
			env:          map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "env-conn", envTelnyxProfileID: "env-profile"},

			expectSource: provider.CredentialSourceEnv, expectHasSecret: true, expectUsable: true,
			expectSecret: fakeEnvTelnyxKey,
			expectConfig: map[string]string{"connection_id": "env-conn", "profile_id": "env-profile"},
		},
		{
			name:         "the config of the row wins over the variable",
			providerName: number.ProviderNameTelnyx,
			env:          map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "env-conn", envTelnyxProfileID: "env-profile"},
			row:          &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Priority: 5, Config: map[string]string{"connection_id": "row-conn"}},

			expectSource: provider.CredentialSourceEnv, expectHasSecret: true, expectUsable: true,
			expectSecret: fakeEnvTelnyxKey,
			expectConfig: map[string]string{"connection_id": "row-conn", "profile_id": "env-profile"},
		},
		{
			name:         "the variable fills an empty config key of the row",
			providerName: number.ProviderNameTelnyx,
			env:          map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "env-conn"},
			row:          &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Priority: 5, Config: map[string]string{"connection_id": ""}},

			expectSource: provider.CredentialSourceEnv, expectHasSecret: true, expectUsable: true,
			expectSecret: fakeEnvTelnyxKey,
			expectConfig: map[string]string{"connection_id": "env-conn"},
		},
		{
			name:         "a placeholder config variable is not set, so the required config is missing",
			providerName: number.ProviderNameTelnyx,
			env:          map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "[[BIN_MANAGER__TELNYX_CONNECTION_ID]]"},

			expectSource: provider.CredentialSourceEnv, expectHasSecret: true, expectUsable: false,
			expectSecret: fakeEnvTelnyxKey,
			expectConfig: map[string]string{},
		},
		{
			name:         "an unset secret is none",
			providerName: number.ProviderNameTelnyx,
			env:          map[string]string{envTelnyxConnectionID: "env-conn"},

			expectSource: provider.CredentialSourceNone, expectHasSecret: false, expectUsable: false,
		},
		{
			name:         "a placeholder secret is none",
			providerName: number.ProviderNameTelnyx,
			env:          map[string]string{envTelnyxAPIKey: "[[BIN_MANAGER__TELNYX_API_KEY]]", envTelnyxConnectionID: "env-conn"},

			expectSource: provider.CredentialSourceNone, expectHasSecret: false, expectUsable: false,
		},
		{
			name:         "the stored ciphertext is not read: a garbage ciphertext without a variable is none",
			providerName: number.ProviderNameTelnyx,
			env:          map[string]string{},
			row:          garbageRow(number.ProviderNameTelnyx, true, map[string]string{"connection_id": "row-conn"}),

			expectSource: provider.CredentialSourceNone, expectHasSecret: false, expectUsable: false,
		},
		{
			name:         "the stored ciphertext is not read: a garbage ciphertext does not affect the variable",
			providerName: number.ProviderNameTelnyx,
			env:          map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey},
			row:          garbageRow(number.ProviderNameTelnyx, true, map[string]string{"connection_id": "row-conn"}),

			expectSource: provider.CredentialSourceEnv, expectHasSecret: true, expectUsable: true,
			expectSecret: fakeEnvTelnyxKey,
			expectConfig: map[string]string{"connection_id": "row-conn"},
		},
		{
			name:         "a disabled row is not usable",
			providerName: number.ProviderNameTelnyx,
			env:          map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "env-conn"},
			row:          &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: false, Priority: 5},

			expectSource: provider.CredentialSourceEnv, expectHasSecret: true, expectUsable: false,
			expectSecret: fakeEnvTelnyxKey,
			expectConfig: map[string]string{"connection_id": "env-conn"},
		},
		{
			name:         "twilio sid comes from the variable, trimmed, and the trunk_sid from the row only",
			providerName: number.ProviderNameTwilio,
			env:          map[string]string{envTwilioAuthToken: fakeEnvTwilioKey, envTwilioAccountSID: "  " + testTwilioSID + "\n"},
			row:          &provider.Provider{ProviderName: number.ProviderNameTwilio, Enabled: true, Priority: 5, Config: map[string]string{"trunk_sid": testTwilioTrunk}},

			expectSource: provider.CredentialSourceEnv, expectHasSecret: true, expectUsable: true,
			expectSecret: fakeEnvTwilioKey,
			expectConfig: map[string]string{"sid": testTwilioSID, "trunk_sid": testTwilioTrunk},
		},
		{
			name:         "twilio without a trunk_sid in the row is not usable, there is no variable for it",
			providerName: number.ProviderNameTwilio,
			env:          map[string]string{envTwilioAuthToken: fakeEnvTwilioKey, envTwilioAccountSID: testTwilioSID},
			row:          &provider.Provider{ProviderName: number.ProviderNameTwilio, Enabled: true, Priority: 5},

			expectSource: provider.CredentialSourceEnv, expectHasSecret: true, expectUsable: false,
			expectSecret: fakeEnvTwilioKey,
			expectConfig: map[string]string{"sid": testTwilioSID},
		},
		{
			name:         "twilio does not read the telnyx variables",
			providerName: number.ProviderNameTwilio,
			env:          map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey},

			expectSource: provider.CredentialSourceNone, expectHasSecret: false, expectUsable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCredentialEnv(t, tt.env)

			db := newFakeDB()
			if tt.row != nil {
				db.rows[tt.providerName] = tt.row
			}
			h := newTestHandler(t, db, "")
			// the stored ciphertext is never decrypted: a call of the crypto would panic
			h.crypto = nil

			r, err := h.resolve(ctx, tt.providerName)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if r.source != tt.expectSource || r.hasSecret != tt.expectHasSecret || r.usable != tt.expectUsable {
				t.Errorf("Wrong match. got source=%s has_secret=%v usable=%v", r.source, r.hasSecret, r.usable)
			}
			if r.creds.Secret != tt.expectSecret {
				t.Errorf("Wrong match. the secret differs (expected one: %v)", tt.expectSecret != "")
			}
			for k, v := range tt.expectConfig {
				if r.creds.Config[k] != v {
					t.Errorf("Wrong match. config key: %s, expect: %q, got: %q", k, v, r.creds.Config[k])
				}
			}
			if len(r.creds.Config) != len(tt.expectConfig) {
				t.Errorf("Wrong match. config: %v, expect: %v", r.creds.Config, tt.expectConfig)
			}
		})
	}
}

func Test_resolve_env_doesNotWriteTheRow(t *testing.T) {
	setCredentialEnv(t, map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "env-conn"})

	db := newFakeDB()
	rowConfig := map[string]string{}
	db.rows[number.ProviderNameTelnyx] = &provider.Provider{ProviderName: number.ProviderNameTelnyx, Enabled: true, Priority: 5, Config: rowConfig}
	h := newTestHandler(t, db, "")

	if _, err := h.resolve(context.Background(), number.ProviderNameTelnyx); err != nil {
		t.Fatal(err)
	}
	if len(rowConfig) != 0 || len(db.rows[number.ProviderNameTelnyx].Config) != 0 {
		t.Errorf("Wrong match. the variable value is in the row config: %v", db.rows[number.ProviderNameTelnyx].Config)
	}
}

func Test_Info_env(t *testing.T) {
	ctx := context.Background()

	t.Run("has_secret follows the effective secret and the config is the row only", func(t *testing.T) {
		setCredentialEnv(t, map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "env-conn"})

		info, err := newTestHandler(t, newFakeDB(), "").Get(ctx, number.ProviderNameTelnyx)
		if err != nil {
			t.Fatal(err)
		}
		if !info.HasSecret || info.CredentialSource != provider.CredentialSourceEnv || !info.Usable {
			t.Errorf("Wrong match. got has_secret=%v source=%s usable=%v", info.HasSecret, info.CredentialSource, info.Usable)
		}
		if len(info.Config) != 0 {
			t.Errorf("Wrong match. the variable value is in the config: %v", info.Config)
		}
	})

	t.Run("a stored ciphertext does not count as a secret", func(t *testing.T) {
		setCredentialEnv(t, map[string]string{})

		db := newFakeDB()
		db.rows[number.ProviderNameTelnyx] = garbageRow(number.ProviderNameTelnyx, true, map[string]string{"connection_id": "row-conn"})

		info, err := newTestHandler(t, db, testKeys(1)).Get(ctx, number.ProviderNameTelnyx)
		if err != nil {
			t.Fatal(err)
		}
		if info.HasSecret || info.CredentialSource != provider.CredentialSourceNone || info.Usable {
			t.Errorf("Wrong match. got has_secret=%v source=%s usable=%v", info.HasSecret, info.CredentialSource, info.Usable)
		}
		if info.Config["connection_id"] != "row-conn" {
			t.Errorf("Wrong match. the config of the row is not shown: %v", info.Config)
		}
	})

	t.Run("a read and write back of the config does not freeze a variable value into the row", func(t *testing.T) {
		setCredentialEnv(t, map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "env-conn", envTelnyxProfileID: "env-profile"})

		db := newFakeDB()
		h := newTestHandler(t, db, "")

		info, err := h.Get(ctx, number.ProviderNameTelnyx)
		if err != nil {
			t.Fatal(err)
		}

		// the admin writes back what it read
		if _, err := h.Update(ctx, number.ProviderNameTelnyx, testActor, boolPtr(true), nil, info.Config); err != nil {
			t.Fatal(err)
		}
		for k, v := range db.rows[number.ProviderNameTelnyx].Config {
			t.Errorf("Wrong match. the row config holds %s=%q", k, v)
		}
	})

	t.Run("deleting the stored secret keeps has_secret true while the variable is set", func(t *testing.T) {
		setCredentialEnv(t, map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "env-conn"})

		db := newFakeDB()
		seedTelnyx(t, db)
		h := newTestHandler(t, db, testKeys(1))

		info, err := h.DeleteSecret(ctx, number.ProviderNameTelnyx, testActor)
		if err != nil {
			t.Fatal(err)
		}
		if !info.HasSecret || info.CredentialSource != provider.CredentialSourceEnv {
			t.Errorf("Wrong match. got has_secret=%v source=%s", info.HasSecret, info.CredentialSource)
		}
	})
}

func Test_PurchaseCandidates_env(t *testing.T) {
	setCredentialEnv(t, map[string]string{envTelnyxAPIKey: fakeEnvTelnyxKey, envTelnyxConnectionID: "env-conn"})

	res, err := newTestHandler(t, newFakeDB(), "").PurchaseCandidates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := candidateNames(res); len(got) != 1 || got[0] != number.ProviderNameTelnyx {
		t.Errorf("Wrong match. expect: [telnyx], got: %v", got)
	}
}

func Test_LogStates(t *testing.T) {
	setCredentialEnv(t, map[string]string{
		envTelnyxAPIKey:       fakeEnvTelnyxKey,
		envTelnyxConnectionID: "[[BIN_MANAGER__TELNYX_CONNECTION_ID]]",
		envTwilioAuthToken:    fakeEnvTwilioKey,
	})

	var buf bytes.Buffer
	orig := logrus.StandardLogger().Out
	logrus.SetOutput(&buf)
	defer logrus.SetOutput(orig)

	LogStates()
	out := buf.String()

	for _, want := range []string{
		"TELNYX_API_KEY", "TELNYX_CONNECTION_ID", "TELNYX_PROFILE_ID", "TWILIO_ACCOUNT_SID", "TWILIO_AUTH_TOKEN",
		"state=set", "state=unset", "state=placeholder", "unresolved placeholder",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("Wrong match. the log misses %q: %s", want, out)
		}
	}
	for _, secret := range []string{fakeEnvTelnyxKey, fakeEnvTwilioKey} {
		if strings.Contains(out, secret) {
			t.Errorf("Wrong match. a value is in the log")
		}
	}
}

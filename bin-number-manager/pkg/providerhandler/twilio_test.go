package providerhandler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
	"monorepo/bin-number-manager/pkg/secretcrypto"
)

const (
	testTwilioSID   = "AC00000000000000000000000000000000"
	testTwilioTrunk = "TK00000000000000000000000000000000"
)

// twilioRow returns a row that holds an encrypted secret and the config.
func twilioRow(t *testing.T, enabled bool, priority int, config map[string]string) *provider.Provider {
	t.Helper()

	crypto, err := secretcrypto.New(testKeys(1))
	if err != nil {
		t.Fatal(err)
	}
	ct, ver, err := crypto.Encrypt("twilio-db-token", string(number.ProviderNameTwilio))
	if err != nil {
		t.Fatal(err)
	}

	return &provider.Provider{
		ProviderName:     number.ProviderNameTwilio,
		Enabled:          enabled,
		Priority:         priority,
		Config:           config,
		SecretCiphertext: ct,
		KeyVersion:       ver,
	}
}

func candidateNames(res []*Candidate) []number.ProviderName {
	got := []number.ProviderName{}
	for _, c := range res {
		got = append(got, c.Name)
	}
	return got
}

func Test_Twilio_purchaseCandidates(t *testing.T) {
	ctx := context.Background()
	full := map[string]string{"sid": testTwilioSID, "trunk_sid": testTwilioTrunk}

	t.Run("deploy default: the environment twilio credential does not change telnyx alone", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", NewEnvCredentials("c", "p", "t", testTwilioSID, "twilio-token"))

		res, err := h.PurchaseCandidates(ctx)
		if err != nil || len(res) != 1 || res[0].Name != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v, %v", candidateNames(res), err)
		}
	})

	t.Run("an enabled twilio with a DB secret and the sid and trunk_sid is a candidate, ordered by priority", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, full)
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		res, err := h.PurchaseCandidates(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got := candidateNames(res); len(got) != 2 || got[0] != number.ProviderNameTwilio || got[1] != number.ProviderNameTelnyx {
			t.Errorf("Wrong order. expect: twilio then telnyx, got: %v", got)
		}
	})

	t.Run("a lower priority twilio stays behind telnyx", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 20, full)
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 2 || got[0] != number.ProviderNameTelnyx || got[1] != number.ProviderNameTwilio {
			t.Errorf("Wrong order. expect: telnyx then twilio, got: %v", got)
		}
	})

	t.Run("a disabled twilio is not a candidate", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, false, 1, full)
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 1 || got[0] != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v", got)
		}
	})

	// the old twilio rows can hold the sid only: they must stay out after the twilio became purchasable
	t.Run("an enabled twilio row with the sid only is not a candidate", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": testTwilioSID})
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 1 || got[0] != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v", got)
		}
	})

	t.Run("an enabled twilio row without any config is not a candidate", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, nil)
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 1 || got[0] != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v", got)
		}
	})

	t.Run("a malformed sid is skipped and counted, telnyx stays", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": "not-a-sid", "trunk_sid": testTwilioTrunk})
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		before := unusableCount(t, "twilio", "factory")
		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 1 || got[0] != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v", got)
		}
		if after := unusableCount(t, "twilio", "factory"); after != before+1 {
			t.Errorf("Wrong metric. expect: %v, got: %v", before+1, after)
		}
	})

	t.Run("a malformed trunk_sid stays a candidate and the purchase refuses it before any request", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": testTwilioSID, "trunk_sid": "bad"})
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 2 || got[0] != number.ProviderNameTwilio {
			t.Fatalf("Wrong match. expect: twilio then telnyx, got: %v", got)
		}
		// the real handler refuses before it touches the network
		if _, err := res[0].Provider.NumberPurchase("+15550000001"); !errors.Is(err, provider.ErrPurchaseRejected) {
			t.Errorf("Wrong match. expect: rejected, got: %v", err)
		}
	})
}

func Test_Twilio_releaseAndSearchNeedTheSidOnly(t *testing.T) {
	ctx := context.Background()

	t.Run("the environment credential can release although it cannot purchase", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", NewEnvCredentials("", "", "", testTwilioSID, "twilio-token"))

		if p, err := h.ForRelease(ctx, number.ProviderNameTwilio); err != nil || p == nil {
			t.Errorf("Wrong match. expect: provider, got: %v, %v", p, err)
		}

		info, err := h.Get(ctx, number.ProviderNameTwilio)
		if err != nil || info.Usable || info.CredentialSource != provider.CredentialSourceEnv {
			t.Errorf("Wrong match. expect: env source and not usable, got: %+v, %v", info, err)
		}
	})

	t.Run("a disabled twilio can release", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, false, 1, map[string]string{"sid": testTwilioSID})
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if p, err := h.ForRelease(ctx, number.ProviderNameTwilio); err != nil || p == nil {
			t.Errorf("Wrong match. expect: provider, got: %v, %v", p, err)
		}
	})

	t.Run("a malformed trunk_sid does not block the release", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": testTwilioSID, "trunk_sid": "bad"})
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if p, err := h.ForRelease(ctx, number.ProviderNameTwilio); err != nil || p == nil {
			t.Errorf("Wrong match. expect: provider, got: %v, %v", p, err)
		}
	})

	t.Run("without the sid the release fails and the error holds no value", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", NewEnvCredentials("", "", "", "", "twilio-token"))

		_, err := h.ForRelease(ctx, number.ProviderNameTwilio)
		if err == nil {
			t.Fatalf("Wrong match. expect: error")
		}
		if msg := err.Error(); len(msg) == 0 || strings.Contains(msg, "twilio-token") {
			t.Errorf("Wrong error. got: %v", err)
		}
	})

	t.Run("a malformed sid fails the release", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", NewEnvCredentials("", "", "", "SK00000000000000000000000000000000", "twilio-token"))

		if _, err := h.ForRelease(ctx, number.ProviderNameTwilio); err == nil {
			t.Errorf("Wrong match. expect: error")
		}
	})

	t.Run("the telnyx environment credential still works with the token alone", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), "", NewEnvCredentials("", "", "env-token", "", ""))

		res, err := h.PurchaseCandidates(ctx)
		if err != nil || len(res) != 1 || res[0].Name != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v, %v", candidateNames(res), err)
		}
	})
}

func Test_Twilio_adminRequiresTheConfigBeforeTheSecret(t *testing.T) {
	ctx := context.Background()

	t.Run("the secret is refused until the sid and trunk_sid are set", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), testKeys(1), envTelnyx())

		if _, err := h.UpdateSecret(ctx, number.ProviderNameTwilio, testActor, "fake-twilio"); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("Wrong match. expect: invalid argument, got: %v", err)
		}
		if _, err := h.Update(ctx, number.ProviderNameTwilio, testActor, nil, nil, map[string]string{"sid": testTwilioSID}); err != nil {
			t.Fatal(err)
		}
		if _, err := h.UpdateSecret(ctx, number.ProviderNameTwilio, testActor, "fake-twilio"); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("Wrong match. the trunk_sid is still missing. expect: invalid argument, got: %v", err)
		}
		if _, err := h.Update(ctx, number.ProviderNameTwilio, testActor, nil, nil, map[string]string{"trunk_sid": testTwilioTrunk}); err != nil {
			t.Fatal(err)
		}
		info, err := h.UpdateSecret(ctx, number.ProviderNameTwilio, testActor, "fake-twilio")
		if err != nil || !info.HasSecret || info.Enabled {
			t.Errorf("Wrong match. expect: stored and disabled, got: %+v, %v", info, err)
		}
	})

	t.Run("the trunk_sid cannot be removed while a secret is stored", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, false, 20, map[string]string{"sid": testTwilioSID, "trunk_sid": testTwilioTrunk})
		h := newTestHandler(t, db, testKeys(1), envTelnyx())

		if _, err := h.Update(ctx, number.ProviderNameTwilio, testActor, nil, nil, map[string]string{"trunk_sid": ""}); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("Wrong match. expect: invalid argument, got: %v", err)
		}
	})

	t.Run("an unknown config key is still refused", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB(), testKeys(1), envTelnyx())

		if _, err := h.Update(ctx, number.ProviderNameTwilio, testActor, nil, nil, map[string]string{"auth_token": "x"}); !errors.Is(err, provider.ErrInvalidArgument) {
			t.Errorf("Wrong match. expect: invalid argument, got: %v", err)
		}
	})
}

func Test_NewEnvCredentials_twilioWhitespace(t *testing.T) {
	env := NewEnvCredentials(" conn ", "prof", " tel-token\n", " "+testTwilioSID+"\n", " twilio-token\r\n")

	tw := env[number.ProviderNameTwilio]
	if tw.Secret != "twilio-token" || tw.Config["sid"] != testTwilioSID {
		t.Errorf("Wrong match. the whitespace of the twilio values must be removed. got: %q, %q", tw.Secret, tw.Config["sid"])
	}

	// the telnyx values are left as they are
	tx := env[number.ProviderNameTelnyx]
	if tx.Secret != " tel-token\n" || tx.Config["connection_id"] != " conn " {
		t.Errorf("Wrong match. the telnyx values must not change. got: %q, %q", tx.Secret, tx.Config["connection_id"])
	}
}

// unusableCount returns the value of provider_unusable_total of the labels.
func unusableCount(t *testing.T, name, reason string) float64 {
	t.Helper()

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range families {
		if f.GetName() != "number_manager_provider_unusable_total" {
			continue
		}
		for _, m := range f.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
			}
			if labels["provider"] == name && labels["reason"] == reason {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

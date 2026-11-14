package providerhandler

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"monorepo/bin-number-manager/models/number"
	"monorepo/bin-number-manager/models/provider"
)

const (
	testTwilioSID   = "AC00000000000000000000000000000000"
	testTwilioTrunk = "TK00000000000000000000000000000000"
)

// twilioRow returns a row that holds the config.
func twilioRow(t *testing.T, enabled bool, priority int, config map[string]string) *provider.Provider {
	t.Helper()

	return &provider.Provider{
		ProviderName: number.ProviderNameTwilio,
		Enabled:      enabled,
		Priority:     priority,
		Config:       config,
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
	setCredentialEnv(t, map[string]string{envTelnyxAPIKey: "env-token", envTwilioAuthToken: "env-twilio-token"})
	full := map[string]string{"sid": testTwilioSID, "trunk_sid": testTwilioTrunk}

	t.Run("without any row there is no candidate: twilio is disabled by default and telnyx has no secret", func(t *testing.T) {
		setCredentialEnv(t, nil)
		h := newTestHandler(t, newFakeDB())

		res, err := h.PurchaseCandidates(ctx)
		if err != nil || len(res) != 0 {
			t.Errorf("Wrong match. expect: none, got: %v, %v", candidateNames(res), err)
		}
	})

	t.Run("an enabled twilio with the variable and the sid and trunk_sid is a candidate, ordered by priority", func(t *testing.T) {
		db := newFakeDB()
		seedTelnyx(t, db)
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, full)
		h := newTestHandler(t, db)

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
		seedTelnyx(t, db)
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 20, full)
		h := newTestHandler(t, db)

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 2 || got[0] != number.ProviderNameTelnyx || got[1] != number.ProviderNameTwilio {
			t.Errorf("Wrong order. expect: telnyx then twilio, got: %v", got)
		}
	})

	t.Run("a disabled twilio is not a candidate", func(t *testing.T) {
		db := newFakeDB()
		seedTelnyx(t, db)
		db.rows[number.ProviderNameTwilio] = twilioRow(t, false, 1, full)
		h := newTestHandler(t, db)

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 1 || got[0] != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v", got)
		}
	})

	// the old twilio rows can hold the sid only: they must stay out after the twilio became purchasable
	t.Run("an enabled twilio row with the sid only is not a candidate", func(t *testing.T) {
		db := newFakeDB()
		seedTelnyx(t, db)
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": testTwilioSID})
		h := newTestHandler(t, db)

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 1 || got[0] != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v", got)
		}
	})

	t.Run("an enabled twilio row without any config is not a candidate", func(t *testing.T) {
		db := newFakeDB()
		seedTelnyx(t, db)
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, nil)
		h := newTestHandler(t, db)

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 1 || got[0] != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v", got)
		}
	})

	t.Run("a malformed sid is skipped and counted, telnyx stays", func(t *testing.T) {
		db := newFakeDB()
		seedTelnyx(t, db)
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": "not-a-sid", "trunk_sid": testTwilioTrunk})
		h := newTestHandler(t, db)

		before := unusableCount(t, "twilio", "factory")
		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 1 || got[0] != number.ProviderNameTelnyx {
			t.Errorf("Wrong match. expect: telnyx only, got: %v", got)
		}
		if after := unusableCount(t, "twilio", "factory"); after != before+1 {
			t.Errorf("Wrong metric. expect: %v, got: %v", before+1, after)
		}
	})

	t.Run("the surrounding whitespace of the stored sid and trunk_sid is removed", func(t *testing.T) {
		db := newFakeDB()
		seedTelnyx(t, db)
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": " " + testTwilioSID + "\n", "trunk_sid": " " + testTwilioTrunk + " "})
		h := newTestHandler(t, db)

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 2 || got[0] != number.ProviderNameTwilio {
			t.Errorf("Wrong match. expect: twilio then telnyx, got: %v", got)
		}
	})

	t.Run("an enabled twilio row without the variable is counted as unusable (no_secret)", func(t *testing.T) {
		setCredentialEnv(t, nil)
		db := newFakeDB()
		seedTelnyx(t, db)
		db.rows[number.ProviderNameTwilio] = &provider.Provider{ProviderName: number.ProviderNameTwilio, Enabled: true, Priority: 1}
		h := newTestHandler(t, db)

		before := unusableCount(t, "twilio", "no_secret")
		_, _ = h.PurchaseCandidates(ctx)
		if after := unusableCount(t, "twilio", "no_secret"); after != before+1 {
			t.Errorf("Wrong metric. expect: %v, got: %v", before+1, after)
		}
	})

	t.Run("a malformed trunk_sid stays a candidate (the purchase itself refuses it, see the numberhandlertwilio tests)", func(t *testing.T) {
		db := newFakeDB()
		seedTelnyx(t, db)
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": testTwilioSID, "trunk_sid": "bad"})
		h := newTestHandler(t, db)

		res, _ := h.PurchaseCandidates(ctx)
		if got := candidateNames(res); len(got) != 2 || got[0] != number.ProviderNameTwilio {
			t.Errorf("Wrong match. expect: twilio then telnyx, got: %v", got)
		}
	})
}

func Test_Twilio_releaseAndSearchNeedTheSidOnly(t *testing.T) {
	ctx := context.Background()
	setCredentialEnv(t, map[string]string{envTwilioAuthToken: "env-twilio-token"})

	t.Run("the variable with the sid only can release although it cannot purchase", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": testTwilioSID})
		h := newTestHandler(t, db)

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
		h := newTestHandler(t, db)

		if p, err := h.ForRelease(ctx, number.ProviderNameTwilio); err != nil || p == nil {
			t.Errorf("Wrong match. expect: provider, got: %v, %v", p, err)
		}
	})

	t.Run("a malformed trunk_sid does not block the release", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": testTwilioSID, "trunk_sid": "bad"})
		h := newTestHandler(t, db)

		if p, err := h.ForRelease(ctx, number.ProviderNameTwilio); err != nil || p == nil {
			t.Errorf("Wrong match. expect: provider, got: %v, %v", p, err)
		}
	})

	t.Run("without the sid the release fails and the error holds no value", func(t *testing.T) {
		h := newTestHandler(t, newFakeDB())

		_, err := h.ForRelease(ctx, number.ProviderNameTwilio)
		if err == nil {
			t.Fatalf("Wrong match. expect: error")
		}
		if msg := err.Error(); len(msg) == 0 || strings.Contains(msg, "twilio-token") {
			t.Errorf("Wrong error. got: %v", err)
		}
	})

	t.Run("a malformed sid fails the release", func(t *testing.T) {
		db := newFakeDB()
		db.rows[number.ProviderNameTwilio] = twilioRow(t, true, 1, map[string]string{"sid": "SK00000000000000000000000000000000"})
		h := newTestHandler(t, db)

		if _, err := h.ForRelease(ctx, number.ProviderNameTwilio); err == nil {
			t.Errorf("Wrong match. expect: error")
		}
	})
}

func Test_Twilio_valuesWithWhitespace(t *testing.T) {
	ctx := context.Background()
	setCredentialEnv(t, map[string]string{envTwilioAuthToken: "  env-twilio-token\r\n", envTwilioAccountSID: " " + testTwilioSID + "\n"})
	db := newFakeDB()
	h := newTestHandler(t, db)

	// the sid of the row is trimmed by the factory, the secret by the variable read
	db.rows[number.ProviderNameTwilio] = &provider.Provider{
		ProviderName: number.ProviderNameTwilio,
		Config:       map[string]string{"sid": " " + testTwilioSID + "\n", "trunk_sid": testTwilioTrunk},
	}
	r, err := h.resolve(ctx, number.ProviderNameTwilio)
	if err != nil || r.creds.Secret != "env-twilio-token" {
		t.Fatalf("Wrong match. the secret must come from the variable, trimmed. got: %v", err)
	}
	if p, err := h.ForRelease(ctx, number.ProviderNameTwilio); err != nil || p == nil {
		t.Errorf("Wrong match. a sid with surrounding whitespace must work. got: %v, %v", p, err)
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

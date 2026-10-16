package aihandler

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

const (
	testCustomModel = ai.EngineModel("customendpoint.my-model")
	testURLA        = "https://llm-a.example.com/v1"
	testURLB        = "https://llm-b.example.com/v1"
	testKeyPlain    = "sentinel-key-MUST-NOT-LEAK-123"
	testURLSecret   = "SENTINEL-URL-PATH-MUST-NOT-LEAK"
)

func newTestCrypto(t *testing.T, versions ...int) *mcpserverhandler.SecretCrypto {
	t.Helper()

	parts := []string{}
	for _, v := range versions {
		raw := bytes.Repeat([]byte{byte(v)}, 32)
		parts = append(parts, fmt.Sprintf("%d:%s", v, base64.StdEncoding.EncodeToString(raw)))
	}

	res, err := mcpserverhandler.NewSecretCrypto(strings.Join(parts, ","))
	if err != nil {
		t.Fatalf("could not build the crypto. err: %v", err)
	}

	return res
}

func strPtr(s string) *string { return &s }

func stubPublicResolver(t *testing.T) {
	t.Helper()

	restore := ai.SetEngineResolverForTest(func(ctx context.Context, host string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	})
	t.Cleanup(restore)
}

func assertReason(t *testing.T, err error, reason string) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("expect a typed error %s. got: %v", reason, err)
	}
	if ve.Reason != reason {
		t.Fatalf("expect reason %s. got: %s", reason, ve.Reason)
	}
	if strings.Contains(err.Error(), "example.com") || strings.Contains(err.Error(), testKeyPlain) || strings.Contains(err.Error(), testURLSecret) {
		t.Errorf("the error carries the url or the key: %v", err)
	}
}

func Test_sealOpenEngineBaseKey(t *testing.T) {
	h := &aiHandler{crypto: newTestCrypto(t, 1, 2)}

	enc, err := h.sealEngineBaseKey(testKeyPlain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Contains(enc, []byte(testKeyPlain)) {
		t.Fatalf("the ciphertext contains the plaintext")
	}
	if int(enc[0])<<8|int(enc[1]) != 2 {
		t.Errorf("expect the highest key version in the first two bytes. got: %v", enc[:2])
	}
	if len(enc) != 2+12+len(testKeyPlain)+16 {
		t.Errorf("unexpected packed length: %d", len(enc))
	}

	got, err := h.openEngineBaseKey(enc)
	if err != nil || got != testKeyPlain {
		t.Fatalf("round trip failed. err: %v", err)
	}

	// an older version still decrypts after a rotation
	old := &aiHandler{crypto: newTestCrypto(t, 1)}
	encOld, err := old.sealEngineBaseKey(testKeyPlain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, err := h.openEngineBaseKey(encOld); err != nil || got != testKeyPlain {
		t.Fatalf("an older version must decrypt. err: %v", err)
	}
	// but a replica lacking the version fails
	if _, err := old.openEngineBaseKey(enc); err == nil {
		t.Errorf("a missing key version must fail")
	}
}

func Test_sealEngineBaseKey_Errors(t *testing.T) {
	t.Run("nil_crypto", func(t *testing.T) {
		h := &aiHandler{}
		if _, err := h.sealEngineBaseKey(testKeyPlain); err == nil {
			t.Fatalf("a nil crypto must fail")
		}
	})

	t.Run("no_key_configured", func(t *testing.T) {
		c, _ := mcpserverhandler.NewSecretCrypto("")
		h := &aiHandler{crypto: c}
		_, err := h.sealEngineBaseKey(testKeyPlain)
		if err == nil || strings.Contains(err.Error(), testKeyPlain) {
			t.Fatalf("expect an error without the key. got: %v", err)
		}
	})

	t.Run("version_above_65535", func(t *testing.T) {
		raw := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
		c, err := mcpserverhandler.NewSecretCrypto("65536:" + raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		h := &aiHandler{crypto: c}
		if _, err := h.sealEngineBaseKey(testKeyPlain); err == nil {
			t.Fatalf("a version above 65535 must fail")
		}

		c2, _ := mcpserverhandler.NewSecretCrypto("65535:" + raw)
		h2 := &aiHandler{crypto: c2}
		if _, err := h2.sealEngineBaseKey(testKeyPlain); err != nil {
			t.Fatalf("65535 must work. err: %v", err)
		}
	})
}

func Test_openEngineBaseKey_Errors(t *testing.T) {
	h := &aiHandler{crypto: newTestCrypto(t, 1)}

	for _, n := range []int{0, 1, 2, 3, 13, 14, 29} {
		if _, err := h.openEngineBaseKey(make([]byte, n)); err == nil {
			t.Errorf("a value of %d bytes must be rejected", n)
		}
	}

	enc, _ := h.sealEngineBaseKey(testKeyPlain)
	tampered := append([]byte{}, enc...)
	tampered[len(tampered)-1] ^= 0xff
	if _, err := h.openEngineBaseKey(tampered); err == nil {
		t.Errorf("a tampered value must be rejected")
	}

	if _, err := (&aiHandler{}).openEngineBaseKey(enc); err == nil {
		t.Errorf("a nil crypto must fail")
	}
}

func Test_FillEngineBaseKey(t *testing.T) {
	h := &aiHandler{crypto: newTestCrypto(t, 1)}
	enc, _ := h.sealEngineBaseKey(testKeyPlain)

	t.Run("fills_a_copy", func(t *testing.T) {
		in := &ai.AI{EngineBaseURL: testURLA, EngineBaseKeyEnc: enc}
		out := h.FillEngineBaseKey(in)

		if string(in.EngineBaseKey) != "" {
			t.Fatalf("the input must not be modified")
		}
		if out == in || string(out.EngineBaseKey) != testKeyPlain || !bytes.Equal(out.EngineBaseKeyEnc, enc) {
			t.Errorf("the copy must carry the plaintext and keep the ciphertext")
		}
	})

	t.Run("no_ciphertext", func(t *testing.T) {
		out := h.FillEngineBaseKey(&ai.AI{EngineBaseKey: "stale"})
		if out.EngineBaseKey != "" {
			t.Errorf("expect an empty plaintext")
		}
	})

	t.Run("nil", func(t *testing.T) {
		if h.FillEngineBaseKey(nil) != nil {
			t.Errorf("expect nil")
		}
	})

	t.Run("decrypt_failure_keeps_the_get_working", func(t *testing.T) {
		other := &aiHandler{crypto: newTestCrypto(t, 9)}
		hook := newSecretLogHook(t)

		in := &ai.AI{EngineBaseURL: testURLA, EngineBaseKeyEnc: enc}
		out := other.FillEngineBaseKey(in)
		if out == nil || out.EngineBaseKey != "" || len(out.EngineBaseKeyEnc) == 0 {
			t.Fatalf("expect empty plaintext and the ciphertext kept")
		}

		errorLogged := false
		for _, e := range hook.AllEntries() {
			if e.Level.String() == "error" {
				errorLogged = true
			}
		}
		if !errorLogged {
			t.Errorf("a decrypt failure must be logged at error level")
		}
		assertNoSecretInLogs(t, hook, "Could not decrypt the engine base key", testKeyPlain, testURLA, string(enc))
	})
}

func Test_publishAIEvent_StripsEndpointKey(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &aiHandler{notifyHandler: mockNotify}

	in := &ai.AI{
		Identity:         commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: uuid.Must(uuid.NewV4())},
		EngineKey:        "legacy-key",
		EngineBaseURL:    testURLA,
		EngineBaseKeyEnc: []byte("ciphertext"),
		EngineBaseKey:    ai.RedactedString(testKeyPlain),
	}

	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), in.CustomerID, ai.EventTypeUpdated, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ string, data any) {
			got := data.(*ai.AI)
			if got.EngineKey != "" || got.EngineBaseKey != "" || got.EngineBaseKeyEnc != nil {
				t.Errorf("the event must carry no key material. got: %+v", got)
			}
			if got.EngineBaseURL != testURLA {
				t.Errorf("the url is not a secret and stays")
			}
		})

	h.publishAIEvent(context.Background(), ai.EventTypeUpdated, in)

	if in.EngineKey == "" || in.EngineBaseKey == "" || in.EngineBaseKeyEnc == nil {
		t.Errorf("the caller's value must keep its key material")
	}
}

type updateHarness struct {
	h      *aiHandler
	mockDB *dbhandler.MockDBHandler
	fields map[ai.Field]any
	called bool
}

func newUpdateHarness(t *testing.T, pre *ai.AI) *updateHarness {
	t.Helper()
	stubPublicResolver(t)

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	res := &updateHarness{
		mockDB: dbhandler.NewMockDBHandler(mc),
	}
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockUtil.EXPECT().UUIDCreate().Return(uuid.Must(uuid.NewV4())).AnyTimes()

	res.h = &aiHandler{
		utilHandler:   mockUtil,
		db:            res.mockDB,
		notifyHandler: mockNotify,
		crypto:        newTestCrypto(t, 1),
	}

	res.mockDB.EXPECT().AIGet(gomock.Any(), pre.ID).Return(pre, nil).AnyTimes()
	res.mockDB.EXPECT().AIUpdate(gomock.Any(), pre.ID, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, f map[ai.Field]any) error {
			res.fields = f
			res.called = true
			return nil
		}).AnyTimes()
	res.mockDB.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	return res
}

// Test_Update_KeyTable covers every row of design section 3.2.
func Test_Update_KeyTable(t *testing.T) {
	storedEnc := []byte("stored-ciphertext")

	tests := []struct {
		name         string
		storedModel  ai.EngineModel
		storedURL    string
		storedEnc    []byte
		model        ai.EngineModel
		url          string
		key          *string
		expectURL    bool
		expectURLVal string
		expectEnc    bool
		expectEncNil bool // when expectEnc: true = NULL, false = sealed value
		expectErr    string
	}{
		{name: "same_url_key_omitted_keeps", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: testURLA, key: nil},
		{name: "same_url_key_empty_deletes", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: testURLA, key: strPtr(""), expectEnc: true, expectEncNil: true},
		{name: "same_url_key_value_replaces", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: testURLA, key: strPtr(testKeyPlain), expectURL: true, expectURLVal: testURLA, expectEnc: true},
		{name: "changed_url_key_omitted_discards", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: testURLB, key: nil, expectURL: true, expectURLVal: testURLB, expectEnc: true, expectEncNil: true},
		{name: "changed_url_key_empty_discards", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: testURLB, key: strPtr(""), expectURL: true, expectURLVal: testURLB, expectEnc: true, expectEncNil: true},
		{name: "changed_url_key_value_replaces", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: testURLB, key: strPtr(testKeyPlain), expectURL: true, expectURLVal: testURLB, expectEnc: true},
		{name: "case_change_counts_as_change", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: "https://LLM-A.example.com/v1", key: nil, expectURL: true, expectURLVal: "https://LLM-A.example.com/v1", expectEnc: true, expectEncNil: true},
		{name: "switch_to_custom_key_omitted", storedModel: ai.EngineModelOpenaiGPT5, storedURL: "", storedEnc: nil,
			model: testCustomModel, url: testURLA, key: nil, expectURL: true, expectURLVal: testURLA, expectEnc: true, expectEncNil: true},
		{name: "switch_to_custom_key_value", storedModel: ai.EngineModelOpenaiGPT5,
			model: testCustomModel, url: testURLA, key: strPtr(testKeyPlain), expectURL: true, expectURLVal: testURLA, expectEnc: true},
		{name: "switch_away_clears_both", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: ai.EngineModelOpenaiGPT5, url: "", key: nil, expectURL: true, expectURLVal: "", expectEnc: true, expectEncNil: true},
		{name: "switch_away_with_empty_key_ok", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: ai.EngineModelOpenaiGPT5, url: "", key: strPtr(""), expectURL: true, expectURLVal: "", expectEnc: true, expectEncNil: true},
		{name: "switch_away_with_url_rejected", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: ai.EngineModelOpenaiGPT5, url: testURLA, key: nil, expectErr: "ENGINE_BASE_URL_NOT_ALLOWED"},
		{name: "switch_away_with_key_rejected", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: ai.EngineModelOpenaiGPT5, url: "", key: strPtr(testKeyPlain), expectErr: "ENGINE_BASE_KEY_NOT_ALLOWED"},
		{name: "non_custom_stays_clears", storedModel: ai.EngineModelOpenaiGPT5,
			model: ai.EngineModelOpenaiGPT5, url: "", key: nil, expectURL: true, expectURLVal: "", expectEnc: true, expectEncNil: true},
		{name: "custom_without_url_rejected", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: "", key: nil, expectErr: "ENGINE_BASE_URL_REQUIRED"},
		{name: "custom_bad_key_rejected", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: testURLA, key: strPtr("bad key"), expectErr: "INVALID_ENGINE_BASE_KEY"},
		{name: "custom_http_url_rejected", storedModel: testCustomModel, storedURL: testURLA, storedEnc: storedEnc,
			model: testCustomModel, url: "http://llm-a.example.com", key: nil, expectErr: "INVALID_ENGINE_BASE_URL"},
	}

	for _, tt := range tests {
		// The three branches of Update: prompt unchanged, changed and cleared.
		for _, pv := range []struct{ suffix, stored, prompt string }{
			{"/prompt_unchanged", "", ""},
			{"/prompt_changed", "", "new prompt"},
			{"/prompt_cleared", "old prompt", ""},
		} {
			prompt := pv.prompt
			name := tt.name + pv.suffix
			t.Run(name, func(t *testing.T) {
				pre := &ai.AI{
					Identity:         commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: uuid.Must(uuid.NewV4())},
					EngineModel:      tt.storedModel,
					EngineBaseURL:    tt.storedURL,
					EngineBaseKeyEnc: tt.storedEnc,
					InitPrompt:       pv.stored,
				}
				u := newUpdateHarness(t, pre)

				// the harness registers AIGet/AIUpdate for pre.ID, so call with it
				_, err := u.h.Update(context.Background(), pre.ID, "n", "", ai.TypeNormal, tt.model, nil, "", tt.url, tt.key, uuid.Nil, prompt,
					ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false)

				if tt.expectErr != "" {
					assertReason(t, err, tt.expectErr)
					if u.called {
						t.Fatalf("a rejected update must not write")
					}
					return
				}
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !u.called {
					t.Fatalf("expect a write")
				}

				gotURL, hasURL := u.fields[ai.FieldEngineBaseURL]
				if hasURL != tt.expectURL {
					t.Fatalf("url column written: expect %v, got %v", tt.expectURL, hasURL)
				}
				if hasURL && gotURL != tt.expectURLVal {
					t.Errorf("url value: expect %q, got %v", tt.expectURLVal, gotURL)
				}

				gotEnc, hasEnc := u.fields[ai.FieldEngineBaseKeyEnc]
				if hasEnc != tt.expectEnc {
					t.Fatalf("enc column written: expect %v, got %v", tt.expectEnc, hasEnc)
				}
				if !hasEnc {
					return
				}
				b, ok := gotEnc.([]byte)
				if !ok {
					t.Fatalf("the ciphertext column must be a []byte (nil is NULL). got: %T", gotEnc)
				}
				if tt.expectEncNil {
					if b != nil {
						t.Errorf("expect NULL")
					}
					return
				}
				plain, errOpen := u.h.openEngineBaseKey(b)
				if errOpen != nil || plain != *tt.key {
					t.Errorf("the sealed key must decrypt to the request value. err: %v", errOpen)
				}
			})
		}
	}
}

func Test_Update_DNSOnlyWhenURLChanges(t *testing.T) {
	tests := []struct {
		name        string
		storedModel ai.EngineModel
		storedURL   string
		url         string
		key         *string
		expectDNS   bool
	}{
		{"unchanged_url_no_dns", testCustomModel, testURLA, testURLA, nil, false},
		{"unchanged_url_key_replace_no_dns", testCustomModel, testURLA, testURLA, strPtr(testKeyPlain), false},
		{"changed_url_dns", testCustomModel, testURLA, testURLB, nil, true},
		{"switch_to_custom_dns", ai.EngineModelOpenaiGPT5, "", testURLA, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pre := &ai.AI{
				Identity:      commonidentity.Identity{ID: uuid.Must(uuid.NewV4()), CustomerID: uuid.Must(uuid.NewV4())},
				EngineModel:   tt.storedModel,
				EngineBaseURL: tt.storedURL,
			}
			u := newUpdateHarness(t, pre)

			lookups := 0
			restore := ai.SetEngineResolverForTest(func(ctx context.Context, host string) ([]netip.Addr, error) {
				lookups++
				return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
			})
			defer restore()

			if _, err := u.h.Update(context.Background(), pre.ID, "n", "", ai.TypeNormal, testCustomModel, nil, "", tt.url, tt.key, uuid.Nil, "",
				ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if (lookups > 0) != tt.expectDNS {
				t.Errorf("dns lookups: expect %v, got %d", tt.expectDNS, lookups)
			}
		})
	}
}

func Test_Update_PrivateHostRejectedWithoutWrite(t *testing.T) {
	pre := &ai.AI{
		Identity:    commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		EngineModel: ai.EngineModelOpenaiGPT5,
	}
	u := newUpdateHarness(t, pre)

	restore := ai.SetEngineResolverForTest(func(ctx context.Context, host string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("10.1.2.3")}, nil
	})
	defer restore()

	_, err := u.h.Update(context.Background(), pre.ID, "n", "", ai.TypeNormal, testCustomModel, nil, "", testURLA, strPtr(testKeyPlain), uuid.Nil, "",
		ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false)
	assertReason(t, err, "ENGINE_BASE_URL_REJECTED")
	if u.called {
		t.Errorf("a rejected update must not write")
	}
}

func Test_Update_NoCryptoFailsWithoutLeak(t *testing.T) {
	pre := &ai.AI{
		Identity:    commonidentity.Identity{ID: uuid.Must(uuid.NewV4())},
		EngineModel: testCustomModel, EngineBaseURL: testURLA,
	}
	u := newUpdateHarness(t, pre)
	u.h.crypto = nil
	hook := newSecretLogHook(t)

	_, err := u.h.Update(context.Background(), pre.ID, "n", "", ai.TypeNormal, testCustomModel, nil, "", testURLA, strPtr(testKeyPlain), uuid.Nil, "",
		ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false)
	if err == nil || u.called {
		t.Fatalf("expect an error and no write. got: %v", err)
	}
	var ve *cerrors.VoipbinError
	if errors.As(err, &ve) {
		t.Errorf("a missing key set is a server error, not a typed 400")
	}
	if strings.Contains(err.Error(), testKeyPlain) {
		t.Errorf("the error carries the key")
	}
	assertNoSecretInLogs(t, hook, "Could not seal the engine base key", testKeyPlain, testURLA)
}

type createHarness struct {
	h       *aiHandler
	created *ai.AI
}

func newCreateHarness(t *testing.T, expectCreate bool) *createHarness {
	t.Helper()
	stubPublicResolver(t)

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	res := &createHarness{}
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)

	res.h = &aiHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify, crypto: newTestCrypto(t, 1)}

	if expectCreate {
		id := uuid.Must(uuid.NewV4())
		mockUtil.EXPECT().UUIDCreate().Return(id).AnyTimes()
		mockReq.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), dmdirect.ResourceTypeAI, id).Return(&dmdirect.Direct{Hash: "h"}, nil)
		mockDB.EXPECT().AICreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, a *ai.AI) error {
			res.created = a
			return nil
		})
		mockDB.EXPECT().AIGet(gomock.Any(), id).Return(&ai.AI{}, nil)
		mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	}
	// without expectations any call to the db, the direct hash or the notifier fails the test

	return res
}

func (c *createHarness) create(model ai.EngineModel, url string, key *string) error {
	_, err := c.h.Create(context.Background(), uuid.Must(uuid.NewV4()), "n", "", ai.TypeNormal, model, nil, "", url, key, uuid.Nil, "",
		ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false)
	return err
}

func Test_Create_Endpoint(t *testing.T) {
	t.Run("custom_with_key", func(t *testing.T) {
		c := newCreateHarness(t, true)
		if err := c.create(testCustomModel, testURLA, strPtr(testKeyPlain)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.created.EngineBaseURL != testURLA {
			t.Errorf("url: %q", c.created.EngineBaseURL)
		}
		if plain, err := c.h.openEngineBaseKey(c.created.EngineBaseKeyEnc); err != nil || plain != testKeyPlain {
			t.Errorf("the stored ciphertext must decrypt to the key. err: %v", err)
		}
		if c.created.EngineBaseKey != "" || c.created.EngineKey != "" {
			t.Errorf("no plaintext may be stored")
		}
	})

	for _, key := range []*string{nil, strPtr("")} {
		t.Run(fmt.Sprintf("custom_without_key_%v", key == nil), func(t *testing.T) {
			c := newCreateHarness(t, true)
			if err := c.create(testCustomModel, testURLA, key); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.created.EngineBaseKeyEnc != nil || c.created.EngineBaseURL != testURLA {
				t.Errorf("expect the url and no ciphertext")
			}
		})
	}

	t.Run("non_custom_stores_nothing", func(t *testing.T) {
		c := newCreateHarness(t, true)
		if err := c.create(ai.EngineModelOpenaiGPT5, "", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.created.EngineBaseURL != "" || c.created.EngineBaseKeyEnc != nil {
			t.Errorf("expect nothing stored")
		}
	})

	rejects := []struct {
		name   string
		model  ai.EngineModel
		url    string
		key    *string
		reason string
	}{
		{"custom_without_url", testCustomModel, "", nil, "ENGINE_BASE_URL_REQUIRED"},
		{"non_custom_with_url", ai.EngineModelOpenaiGPT5, testURLA, nil, "ENGINE_BASE_URL_NOT_ALLOWED"},
		{"non_custom_with_key", ai.EngineModelOpenaiGPT5, "", strPtr(testKeyPlain), "ENGINE_BASE_KEY_NOT_ALLOWED"},
		{"bad_key", testCustomModel, testURLA, strPtr("a b"), "INVALID_ENGINE_BASE_KEY"},
		{"http_url", testCustomModel, "http://llm.example.com/" + testURLSecret, nil, "INVALID_ENGINE_BASE_URL"},
		{"url_with_credentials", testCustomModel, "https://user:" + testURLSecret + "@llm.example.com", nil, "INVALID_ENGINE_BASE_URL"},
		{"single_label_host", testCustomModel, "https://intranet/" + testURLSecret, nil, "INVALID_ENGINE_BASE_URL"},
		{"literal_private", testCustomModel, "https://10.0.0.1/" + testURLSecret, nil, "ENGINE_BASE_URL_REJECTED"},
		{"bad_model_id", "customendpoint.", "", nil, "INVALID_ENGINE_MODEL"},
	}
	for _, tt := range rejects {
		t.Run("rejects_"+tt.name, func(t *testing.T) {
			c := newCreateHarness(t, false) // nothing may be created, not even the direct hash
			assertReason(t, c.create(tt.model, tt.url, tt.key), tt.reason)
		})
	}

	t.Run("rejects_private_resolution", func(t *testing.T) {
		c := newCreateHarness(t, false)
		restore := ai.SetEngineResolverForTest(func(ctx context.Context, host string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("192.168.1.1")}, nil
		})
		defer restore()
		assertReason(t, c.create(testCustomModel, testURLA, strPtr(testKeyPlain)), "ENGINE_BASE_URL_REJECTED")
	})

	t.Run("no_crypto_creates_nothing", func(t *testing.T) {
		c := newCreateHarness(t, false)
		c.h.crypto = nil
		err := c.create(testCustomModel, testURLA, strPtr(testKeyPlain))
		if err == nil || strings.Contains(err.Error(), testKeyPlain) {
			t.Fatalf("expect an error without the key. got: %v", err)
		}
	})
}

func Test_Create_Update_LogsCarryNoSecret(t *testing.T) {
	hook := newSecretLogHook(t)

	c := newCreateHarness(t, true)
	url := "https://llm.example.com/" + testURLSecret
	if err := c.create(testCustomModel, url, strPtr(testKeyPlain)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// rejected path
	c2 := newCreateHarness(t, false)
	_ = c2.create(testCustomModel, "https://10.0.0.1/"+testURLSecret, strPtr(testKeyPlain))

	pre := &ai.AI{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, EngineModel: testCustomModel, EngineBaseURL: url}
	u := newUpdateHarness(t, pre)
	if _, err := u.h.Update(context.Background(), pre.ID, "n", "", ai.TypeNormal, testCustomModel, nil, "", url, strPtr(testKeyPlain), uuid.Nil, "new prompt",
		ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, e := range hook.AllEntries() {
		for _, secret := range []string{testKeyPlain, testURLSecret} {
			if strings.Contains(e.Message, secret) || strings.Contains(fmt.Sprintf("%v", e.Data), secret) {
				t.Errorf("secret leaked into the log entry %q", e.Message)
			}
		}
	}
}

func Test_Get_List_DoNotDecrypt(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &aiHandler{db: mockDB, crypto: newTestCrypto(t, 1)}
	enc, _ := h.sealEngineBaseKey(testKeyPlain)
	stored := &ai.AI{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, EngineBaseKeyEnc: enc}

	mockDB.EXPECT().AIGet(gomock.Any(), stored.ID).Return(stored, nil)
	mockDB.EXPECT().AIList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*ai.AI{stored}, nil)

	got, err := h.Get(context.Background(), stored.ID)
	if err != nil || got.EngineBaseKey != "" {
		t.Errorf("Get must not decrypt. err: %v", err)
	}
	list, err := h.List(context.Background(), 10, "", nil)
	if err != nil || list[0].EngineBaseKey != "" {
		t.Errorf("List must not decrypt. err: %v", err)
	}
}

func Test_Delete_PublishesNoKeyMaterial(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &aiHandler{db: mockDB, notifyHandler: mockNotify}

	stored := &ai.AI{Identity: commonidentity.Identity{ID: uuid.Must(uuid.NewV4())}, EngineBaseURL: testURLA, EngineBaseKeyEnc: []byte("ciphertext")}
	mockDB.EXPECT().AIGet(gomock.Any(), stored.ID).Return(stored, nil).Times(2)
	mockDB.EXPECT().AIDelete(gomock.Any(), stored.ID).Return(nil)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), ai.EventTypeDeleted, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ uuid.UUID, _ string, data any) {
			if data.(*ai.AI).EngineBaseKeyEnc != nil {
				t.Errorf("the delete event must carry no ciphertext")
			}
		})

	if _, err := h.Delete(context.Background(), stored.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

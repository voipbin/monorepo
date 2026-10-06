package auth

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"

	"github.com/gofrs/uuid"
	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
)

const (
	sentinelDirectHash  = "sentinel-direct-hash-5e1a"
	sentinelPasswordH   = "sentinel-password-hash-77c2"
	sentinelRawToken    = "sentinel-raw-token-91bd"
	sentinelFingerprint = "sentinel-hash-fingerprint-3f0a"
)

func newMarshalTestIdentities() map[string]*AuthIdentity {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	return map[string]*AuthIdentity{
		"agent": {
			Type:       TypeAgent,
			CustomerID: customerID,
			Agent: &amagent.Agent{
				Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"), CustomerID: customerID},
				Username:     "agent-user",
				Name:         "agent name",
				Status:       amagent.StatusAvailable,
				Permission:   amagent.PermissionCustomerAdmin,
				PasswordHash: sentinelPasswordH,
				DirectHash:   sentinelDirectHash,
				DirectID:     uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			},
		},
		"accesskey": {
			Type:       TypeAccesskey,
			CustomerID: customerID,
			Accesskey: &csaccesskey.Accesskey{
				ID:          uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
				CustomerID:  customerID,
				TokenHash:   "sentinel-token-hash",
				TokenPrefix: "abcd1234",
				RawToken:    sentinelRawToken,
			},
		},
		"direct": {
			Type:       TypeDirect,
			CustomerID: customerID,
			DirectScope: &DirectScope{
				CustomerID:           customerID,
				ResourceType:         "extension",
				ResourceID:           uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
				AllowedResourceTypes: []string{"call"},
				DirectID:             uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
				HashFingerprint:      sentinelFingerprint,
				BootExpire:           "2026-10-07T00:00:00.000000Z",
				ScopeVersion:         2,
			},
		},
		"delegate": {
			Type:       TypeDelegate,
			CustomerID: customerID,
			DelegateScope: &DelegateScope{
				CustomerID: customerID,
				IssuedBy:   uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
				JTI:        "jti-1",
			},
		},
	}
}

var marshalSecrets = []string{sentinelDirectHash, sentinelPasswordH, sentinelRawToken, sentinelFingerprint, "sentinel-token-hash"}

func Test_AuthIdentity_MarshalJSON_noSecrets(t *testing.T) {
	for name, id := range newMarshalTestIdentities() {
		t.Run(name, func(t *testing.T) {
			for _, form := range []any{id, *id} {
				b, err := json.Marshal(form)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				for _, s := range marshalSecrets {
					if strings.Contains(string(b), s) {
						t.Errorf("secret %q leaked: %s", s, b)
					}
				}
			}
		})
	}
}

func Test_AuthIdentity_MarshalJSON_shape(t *testing.T) {
	ids := newMarshalTestIdentities()

	tests := []struct {
		name       string
		member     string
		wantKeys   []string
		wantNoKey  []string
		wantValues map[string]string
	}{
		{"agent", "Agent", []string{"id", "customer_id", "username", "name", "status", "permission"}, []string{"direct_hash", "direct_id", "addresses", "tag_ids", "ring_method", "detail", "password_hash"}, map[string]string{"id": "d152e69e-105b-11ee-b395-eb18426de979", "customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c", "username": "agent-user", "name": "agent name"}},
		{"accesskey", "Accesskey", []string{"id", "customer_id", "token_prefix"}, []string{"raw_token", "token_hash", "name", "detail"}, map[string]string{"id": "22222222-2222-2222-2222-222222222222", "customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c", "token_prefix": "abcd1234"}},
		{"direct", "DirectScope", []string{"customer_id", "resource_type", "resource_id", "allowed_resource_types", "allowed_resource_id", "direct_id", "boot_expire", "scope_version"}, []string{"hash_fingerprint"}, map[string]string{"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c", "resource_type": "extension", "resource_id": "33333333-3333-3333-3333-333333333333", "direct_id": "44444444-4444-4444-4444-444444444444", "boot_expire": "2026-10-07T00:00:00.000000Z"}},
		{"delegate", "DelegateScope", []string{"customer_id", "issued_by", "jti"}, nil, map[string]string{"issued_by": "55555555-5555-5555-5555-555555555555", "jti": "jti-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(ids[tt.name])
			if err != nil {
				t.Fatal(err)
			}
			var top map[string]json.RawMessage
			if err := json.Unmarshal(b, &top); err != nil {
				t.Fatal(err)
			}
			for _, k := range []string{"Type", "CustomerID", "Agent", "Accesskey", "DirectScope", "DelegateScope"} {
				if _, ok := top[k]; !ok {
					t.Errorf("top-level key %q missing: %s", k, b)
				}
			}
			for _, other := range []string{"Agent", "Accesskey", "DirectScope", "DelegateScope"} {
				if other != tt.member && string(top[other]) != "null" {
					t.Errorf("member %q should be null, got %s", other, top[other])
				}
			}
			var m map[string]any
			if err := json.Unmarshal(top[tt.member], &m); err != nil {
				t.Fatalf("member %s: %v (%s)", tt.member, err, top[tt.member])
			}
			for _, k := range tt.wantKeys {
				if _, ok := m[k]; !ok {
					t.Errorf("key %q missing in %s: %s", k, tt.member, top[tt.member])
				}
			}
			for k, want := range tt.wantValues {
				if got, _ := m[k].(string); got != want {
					t.Errorf("value of %q in %s: got %q, want %q", k, tt.member, got, want)
				}
			}
			var topCustomerID, topType string
			_ = json.Unmarshal(top["CustomerID"], &topCustomerID)
			_ = json.Unmarshal(top["Type"], &topType)
			if topCustomerID != "5f621078-8e5f-11ee-97b2-cfe7337b701c" || topType != tt.name {
				t.Errorf("top-level values: type %q, customer id %q", topType, topCustomerID)
			}
			for _, k := range tt.wantNoKey {
				if _, ok := m[k]; ok {
					t.Errorf("key %q must be absent in %s", k, tt.member)
				}
			}
		})
	}
}

func Test_AuthIdentity_MarshalJSON_nilPointer(t *testing.T) {
	var id *AuthIdentity
	b, err := json.Marshal(id)
	if err != nil || string(b) != "null" {
		t.Errorf("nil pointer: got %s, err %v", b, err)
	}
}

func Test_AuthIdentity_logFormatter_noSecrets(t *testing.T) {
	for name, id := range newMarshalTestIdentities() {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			l := logrus.New()
			l.SetOutput(&buf)
			l.SetFormatter(joonix.NewFormatter())
			l.SetLevel(logrus.DebugLevel)
			l.WithField("agent", id).Debug("test")
			l.WithFields(logrus.Fields{"auth": id, "auth_identity": id}).Debug("test")

			if buf.Len() == 0 {
				t.Fatal("no log output")
			}
			for _, s := range marshalSecrets {
				if strings.Contains(buf.String(), s) {
					t.Errorf("secret %q leaked into log output: %s", s, buf.String())
				}
			}
		})
	}
}

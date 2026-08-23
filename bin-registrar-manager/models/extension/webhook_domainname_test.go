package extension

import (
	"encoding/json"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Test_ConvertWebhookMessage_domainNameServingRule covers the VOIP-1385
// post-cutover serving rule at the webhook conversion point: serve Realm when
// set; serve the stored DomainName as-is only when Realm is empty
// (deleted-customer orphan rows).
func Test_ConvertWebhookMessage_domainNameServingRule(t *testing.T) {

	customerID := uuid.FromStringOrNil("a0b1c2d3-7f84-11ee-8f5a-1b2c3d4e5f60")

	tests := []struct {
		name string

		ext *Extension

		expectDomainName string
	}{
		{
			name: "realm set: realm wins over the stored domain_name",

			ext: &Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				DomainName: customerID.String(), // stale stored value
				Realm:      "ab12.reg.voipbin.net",
			},

			expectDomainName: "ab12.reg.voipbin.net",
		},
		{
			name: "realm set and equal to domain_name (post-batch row)",

			ext: &Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				DomainName: "cd34.reg.voipbin.net",
				Realm:      "cd34.reg.voipbin.net",
			},

			expectDomainName: "cd34.reg.voipbin.net",
		},
		{
			name: "realm empty (orphan row): stored domain_name is served as-is",

			ext: &Extension{
				Identity:   commonidentity.Identity{CustomerID: customerID},
				DomainName: customerID.String(), // whatever is stored; representation no longer matters
				Realm:      "",
			},

			expectDomainName: customerID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wm := tt.ext.ConvertWebhookMessage()

			if wm.DomainName != tt.expectDomainName {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectDomainName, wm.DomainName)
			}
		})
	}
}

// Test_CreateWebhookEvent_domainNameServingRule verifies the serialized
// webhook payload carries the served value, not the raw stored one.
func Test_CreateWebhookEvent_domainNameServingRule(t *testing.T) {
	customerID := uuid.FromStringOrNil("b1c2d3e4-7f84-11ee-8f5a-1b2c3d4e5f60")

	ext := &Extension{
		Identity:   commonidentity.Identity{CustomerID: customerID},
		DomainName: customerID.String(),
		Realm:      "ef56.reg.voipbin.net",
	}

	data, err := ext.CreateWebhookEvent()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	payload := map[string]any{}
	if errUnmarshal := json.Unmarshal(data, &payload); errUnmarshal != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errUnmarshal)
	}

	if payload["domain_name"] != "ef56.reg.voipbin.net" {
		t.Errorf("Wrong match. expect: ef56.reg.voipbin.net, got: %v", payload["domain_name"])
	}
	if strings.Contains(string(data), `"domain_name":"`+customerID.String()+`"`) {
		t.Errorf("Wrong match. raw bare-uuid domain_name leaked into the webhook payload: %s", data)
	}
}

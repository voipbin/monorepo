package aihandler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
)

func Test_publishAIEvent_stripsEngineKey(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000001")

	tests := []struct {
		name      string
		eventType string
	}{
		{"created", ai.EventTypeCreated},
		{"updated", ai.EventTypeUpdated},
		{"deleted", ai.EventTypeDeleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &aiHandler{notifyHandler: mockNotify}

			orig := &ai.AI{
				Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000001"), CustomerID: customerID},
				Name:        "test",
				EngineModel: "openrouter.vendor/model",
				EngineKey:   "dummy-key-value",
			}

			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, tt.eventType, gomock.Any()).
				Do(func(_ context.Context, _ uuid.UUID, _ string, data notifyhandler.WebhookEventMessage) {
					b, err := json.Marshal(data)
					if err != nil {
						t.Fatalf("marshal: %v", err)
					}
					if strings.Contains(string(b), "dummy-key-value") || strings.Contains(string(b), `"engine_key"`) {
						t.Errorf("published payload must not carry the engine key: %s", b)
					}
					if !strings.Contains(string(b), `"has_engine_key":true`) {
						t.Errorf("the event must report that a key is held: %s", b)
					}
					hook, err := data.CreateWebhookEvent()
					if err != nil {
						t.Fatalf("webhook payload: %v", err)
					}
					if !strings.Contains(string(hook), `"has_engine_key":true`) || strings.Contains(string(hook), "dummy-key-value") {
						t.Errorf("webhook payload must report the key and not carry it: %s", hook)
					}
					if !strings.Contains(string(b), "openrouter.vendor/model") {
						t.Errorf("other fields must be kept: %s", b)
					}
				}).Times(1)

			h.publishAIEvent(context.Background(), tt.eventType, orig)

			if orig.EngineKey != "dummy-key-value" {
				t.Errorf("the original must keep its engine key, got %q", orig.EngineKey)
			}
		})
	}
}

func Test_publishAIEvent_noKeyReportsFalse(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1c1c1c1-0000-0000-0000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &aiHandler{notifyHandler: mockNotify}

	orig := &ai.AI{
		Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("a1a1a1a1-0000-0000-0000-000000000001"), CustomerID: customerID},
		EngineModel: "anthropic.claude-haiku-4.5",
	}

	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, ai.EventTypeUpdated, gomock.Any()).
		Do(func(_ context.Context, _ uuid.UUID, _ string, data notifyhandler.WebhookEventMessage) {
			hook, err := data.CreateWebhookEvent()
			if err != nil {
				t.Fatalf("webhook payload: %v", err)
			}
			if !strings.Contains(string(hook), `"has_engine_key":false`) {
				t.Errorf("an AI without a key must report false: %s", hook)
			}
		}).Times(1)

	h.publishAIEvent(context.Background(), ai.EventTypeUpdated, orig)

	if orig.HasEngineKey {
		t.Errorf("the caller's value must not get the hint")
	}
}

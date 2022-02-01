package dbhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/messagetarget"
	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/cachehandler"
)

func TestMessageTargetSet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		message *messagetarget.MessageTarget
	}{
		{
			"normal",
			&messagetarget.MessageTarget{
				ID:            uuid.FromStringOrNil("2608104c-833b-11ec-96be-3f85e20de743"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().MessageTargetSet(gomock.Any(), gomock.Any()).Return(nil)

			if err := h.MessageTargetSet(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func TestMessageTargetGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	tests := []struct {
		name string

		id        uuid.UUID
		expectRes *messagetarget.MessageTarget
	}{
		{
			"normal",
			uuid.FromStringOrNil("7139cd76-833b-11ec-8037-3b32f3ee8e07"),
			&messagetarget.MessageTarget{
				ID:            uuid.FromStringOrNil("7139cd76-833b-11ec-8037-3b32f3ee8e07"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockCache.EXPECT().MessageTargetGet(gomock.Any(), tt.id).Return(tt.expectRes, nil)

			res, err := h.MessageTargetGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

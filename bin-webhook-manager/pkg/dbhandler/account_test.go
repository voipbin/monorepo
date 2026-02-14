package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webhook-manager/models/account"
	"monorepo/bin-webhook-manager/pkg/cachehandler"
)

func Test_AccountSet(t *testing.T) {
	tests := []struct {
		name string

		message *account.Account
	}{
		{
			"normal",
			&account.Account{
				ID:            uuid.FromStringOrNil("2608104c-833b-11ec-96be-3f85e20de743"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().AccountSet(gomock.Any(), gomock.Any()).Return(nil)

			if err := h.AccountSet(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_AccountGet(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		expectRes *account.Account
	}{
		{
			"normal",
			uuid.FromStringOrNil("7139cd76-833b-11ec-8037-3b32f3ee8e07"),
			&account.Account{
				ID:            uuid.FromStringOrNil("7139cd76-833b-11ec-8037-3b32f3ee8e07"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().AccountGet(gomock.Any(), tt.id).Return(tt.expectRes, nil)

			res, err := h.AccountGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}

		})
	}
}

func Test_AccountGetError(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID
	}{
		{
			"cache_error",
			uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().AccountGet(gomock.Any(), tt.id).Return(nil, ErrNotFound)

			res, err := h.AccountGet(ctx, tt.id)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_AccountSetError(t *testing.T) {
	tests := []struct {
		name    string
		message *account.Account
	}{
		{
			"cache_error",
			&account.Account{
				ID:            uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
				WebhookMethod: "POST",
				WebhookURI:    "test.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().AccountSet(gomock.Any(), gomock.Any()).Return(fmt.Errorf("cache error"))

			if err := h.AccountSet(ctx, tt.message); err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_NewHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)
	h := NewHandler(dbTest, mockCache)

	if h == nil {
		t.Errorf("Wrong match. expect: handler, got: nil")
	}
}

package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

func TestAstAuthCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name       string
		auth       *astauth.AstAuth
		expectAuth *astauth.AstAuth
	}

	tests := []test{
		{
			"test normal",
			&astauth.AstAuth{
				ID:       getStringPointer("test1@test.sip.voipbin.net"),
				AuthType: getStringPointer("userpass"),

				Username: getStringPointer("test1"),
				Password: getStringPointer("password"),

				Realm: getStringPointer("test.sip.voipbin.net"),
			},
			&astauth.AstAuth{
				ID:       getStringPointer("test1@test.sip.voipbin.net"),
				AuthType: getStringPointer("userpass"),

				Username: getStringPointer("test1"),
				Password: getStringPointer("password"),

				Realm: getStringPointer("test.sip.voipbin.net"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().AstAuthSet(gomock.Any(), gomock.Any())
			if err := h.AstAuthCreate(context.Background(), tt.auth); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstAuthGet(gomock.Any(), *tt.auth.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AstAuthSet(gomock.Any(), gomock.Any())
			res, err := h.AstAuthGet(context.Background(), *tt.auth.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectAuth, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectAuth, res)
			}
		})
	}
}

func TestAstAuthDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string
		auth *astauth.AstAuth
	}

	tests := []test{
		{
			"test normal",
			&astauth.AstAuth{
				ID:       getStringPointer("dcd14fe0-6df8-11eb-96b2-9f307c0f50bf@test.sip.voipbin.net"),
				AuthType: getStringPointer("userpass"),

				Username: getStringPointer("dcd14fe0-6df8-11eb-96b2-9f307c0f50bf"),
				Password: getStringPointer("password"),

				Realm: getStringPointer("test.sip.voipbin.net"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().AstAuthSet(gomock.Any(), gomock.Any())
			if err := h.AstAuthCreate(context.Background(), tt.auth); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstAuthDel(gomock.Any(), *tt.auth.ID)
			if err := h.AstAuthDelete(context.Background(), *tt.auth.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstAuthGet(gomock.Any(), *tt.auth.ID).Return(nil, fmt.Errorf(""))
			_, err := h.AstAuthGet(context.Background(), *tt.auth.ID)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func TestAstAuthUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name       string
		auth       *astauth.AstAuth
		updateAuth *astauth.AstAuth
		expectAuth *astauth.AstAuth
	}

	tests := []test{
		{
			"test normal",
			&astauth.AstAuth{
				ID:       getStringPointer("fc48baa8-6f41-11eb-9209-ff1f20a9494e"),
				AuthType: getStringPointer("userpass"),
				Username: getStringPointer("test"),
				Password: getStringPointer("password"),
			},
			&astauth.AstAuth{
				ID:       getStringPointer("fc48baa8-6f41-11eb-9209-ff1f20a9494e"),
				Password: getStringPointer("update password"),
			},
			&astauth.AstAuth{
				ID:       getStringPointer("fc48baa8-6f41-11eb-9209-ff1f20a9494e"),
				AuthType: getStringPointer("userpass"),
				Username: getStringPointer("test"),
				Password: getStringPointer("update password"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().AstAuthSet(gomock.Any(), gomock.Any())
			if err := h.AstAuthCreate(ctx, tt.auth); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstAuthSet(gomock.Any(), gomock.Any())
			if err := h.AstAuthUpdate(ctx, tt.updateAuth); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstAuthGet(gomock.Any(), *tt.updateAuth.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AstAuthSet(gomock.Any(), gomock.Any())
			res, err := h.AstAuthGet(context.Background(), *tt.updateAuth.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectAuth, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectAuth, res)
			}
		})
	}
}

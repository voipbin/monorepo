package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

func TestAstAORCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name      string
		aor       *astaor.AstAOR
		expectAOR *astaor.AstAOR
	}

	tests := []test{
		{
			"test normal",
			&astaor.AstAOR{
				ID:          getStringPointer("test1@test.sip.voipbin.net"),
				MaxContacts: getIntegerPointer(1),
			},
			&astaor.AstAOR{
				ID:          getStringPointer("test1@test.sip.voipbin.net"),
				MaxContacts: getIntegerPointer(1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().AstAORSet(gomock.Any(), gomock.Any())
			if err := h.AstAORCreate(context.Background(), tt.aor); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstAORGet(gomock.Any(), *tt.aor.ID.Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AstAORSet(gomock.Any(), gomock.Any())
			res, err := h.AstAORGet(context.Background(), *tt.aor.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectAOR, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectAOR, res)
			}
		})
	}
}

func TestAstAORDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string
		aor  *astaor.AstAOR
	}

	tests := []test{
		{
			"test normal",
			&astaor.AstAOR{
				ID:          getStringPointer("05fb910e-6e04-11eb-b8f6-73501fb47ab3@test.sip.voipbin.net"),
				MaxContacts: getIntegerPointer(1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().AstAORSet(gomock.Any(), gomock.Any())
			if err := h.AstAORCreate(context.Background(), tt.aor); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstAORDel(gomock.Any(), *tt.aor.ID)
			if err := h.AstAORDelete(context.Background(), *tt.aor.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstAORGet(gomock.Any(), *tt.aor.ID.Return(nil, fmt.Errorf(""))
			_, err := h.AstAORGet(context.Background(), *tt.aor.ID)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
		})
	}
}

package dbhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

func TestAstEndpointCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name           string
		endpoint       *astendpoint.AstEndpoint
		expectEndpoint *astendpoint.AstEndpoint
	}

	tests := []test{
		{
			"test normal",
			&astendpoint.AstEndpoint{
				ID:        getStringPointer("test1@test.sip.voipbin.net"),
				AORs:      getStringPointer("test1@test.sip.voipbin.net"),
				Auth:      getStringPointer("test1@test.sip.voipbin.net"),
				Transport: getStringPointer("transport-tcp"),
			},
			&astendpoint.AstEndpoint{
				ID:        getStringPointer("test1@test.sip.voipbin.net"),
				AORs:      getStringPointer("test1@test.sip.voipbin.net"),
				Auth:      getStringPointer("test1@test.sip.voipbin.net"),
				Transport: getStringPointer("transport-tcp"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().AstEndpointSet(gomock.Any(), gomock.Any())
			if err := h.AstEndpointCreate(context.Background(), tt.endpoint); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstEndpointGet(gomock.Any(), *tt.endpoint.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().AstEndpointSet(gomock.Any(), gomock.Any())
			res, err := h.AstEndpointGet(context.Background(), *tt.endpoint.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectEndpoint, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectEndpoint, res)
			}
		})
	}
}

func TestAstEndpointDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name     string
		endpoint *astendpoint.AstEndpoint
	}

	tests := []test{
		{
			"test normal",
			&astendpoint.AstEndpoint{
				ID:        getStringPointer("0f1e4e7e-6e00-11eb-91a1-9f89915551df@test.sip.voipbin.net"),
				AORs:      getStringPointer("0f1e4e7e-6e00-11eb-91a1-9f89915551df@test.sip.voipbin.net"),
				Auth:      getStringPointer("0f1e4e7e-6e00-11eb-91a1-9f89915551df@test.sip.voipbin.net"),
				Transport: getStringPointer("transport-tcp"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().AstEndpointSet(gomock.Any(), gomock.Any())
			if err := h.AstEndpointCreate(context.Background(), tt.endpoint); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstEndpointDel(gomock.Any(), *tt.endpoint.ID)
			if err := h.AstEndpointDelete(context.Background(), *tt.endpoint.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().AstEndpointGet(gomock.Any(), *tt.endpoint.ID).Return(nil, fmt.Errorf(""))
			_, err := h.AstEndpointGet(context.Background(), *tt.endpoint.ID)
			if err == nil {
				t.Errorf("Wrong match. expect: err, got: ok")
			}
		})
	}
}

package outboundconfighandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	gomock "go.uber.org/mock/gomock"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_NewOutboundConfigHandler_cov2(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	res := NewOutboundConfigHandler(mockUtil, mockDB, mockCache, mockReq)

	expect := &outboundConfigHandler{utilHandler: mockUtil, db: mockDB, cacheHandler: mockCache, reqHandler: mockReq}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

func Test_outboundConfigHandler_Delete_cov2_dbDeleteError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &outboundConfigHandler{db: mockDB}
	ctx := context.Background()
	id := uuid.FromStringOrNil("66666666-0000-0000-0000-000000000001")

	mockDB.EXPECT().OutboundConfigGetByID(ctx, id).Return(&outboundconfig.OutboundConfig{ID: id}, nil)
	dbErr := errors.New("delete failed")
	mockDB.EXPECT().OutboundConfigDelete(ctx, id).Return(dbErr)

	res, err := h.Delete(ctx, id)
	if res != nil || !errors.Is(err, dbErr) {
		t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", dbErr, res, err)
	}
}

func Test_outboundConfigHandler_GetByCustomerID_cov2_cacheErrorFallsThrough(t *testing.T) {
	tests := []struct {
		name string

		cacheErr error
	}{
		{name: "redis nil", cacheErr: redis.Nil},
		{name: "other cache error", cacheErr: errors.New("cache down")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &outboundConfigHandler{db: mockDB, cacheHandler: mockCache}
			ctx := context.Background()
			customerID := uuid.FromStringOrNil("66666666-0000-0000-0000-0000000000cc")
			row := &outboundconfig.OutboundConfig{ID: uuid.FromStringOrNil("66666666-0000-0000-0000-000000000001"), CustomerID: customerID, Name: "n"}

			mockCache.EXPECT().OutboundConfigGet(ctx, customerID).Return(nil, tt.cacheErr)
			mockDB.EXPECT().OutboundConfigGetByCustomerID(ctx, customerID).Return(row, nil)
			mockCache.EXPECT().OutboundConfigSet(ctx, customerID, row).Return(nil)

			res, err := h.GetByCustomerID(ctx, customerID)
			if err != nil || !reflect.DeepEqual(res, row) {
				t.Errorf("Wrong match. expect: %v, got: %v, err: %v", row, res, err)
			}
		})
	}
}

func Test_applyUpdateRequest_cov2(t *testing.T) {
	detail := "new detail"
	name := "new name"
	whitelist := []string{"+82"}
	codecs := "ulaw"
	numberID := uuid.FromStringOrNil("66666666-0000-0000-0000-0000000000aa")

	tests := []struct {
		name string

		req *outboundconfig.UpdateRequest

		expect outboundconfig.OutboundConfig
	}{
		{
			name:   "nil request leaves the config untouched",
			req:    nil,
			expect: outboundconfig.OutboundConfig{Name: "old", Detail: "old detail"},
		},
		{
			name:   "empty request leaves the config untouched",
			req:    &outboundconfig.UpdateRequest{},
			expect: outboundconfig.OutboundConfig{Name: "old", Detail: "old detail"},
		},
		{
			name:   "detail only",
			req:    &outboundconfig.UpdateRequest{Detail: &detail},
			expect: outboundconfig.OutboundConfig{Name: "old", Detail: detail},
		},
		{
			name: "all fields",
			req: &outboundconfig.UpdateRequest{
				Name: &name, Detail: &detail, DestinationWhitelist: &whitelist, Codecs: &codecs, DefaultOutgoingSourceNumberID: &numberID,
			},
			expect: outboundconfig.OutboundConfig{
				Name: name, Detail: detail, DestinationWhitelist: whitelist, Codecs: codecs, DefaultOutgoingSourceNumberID: numberID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &outboundconfig.OutboundConfig{Name: "old", Detail: "old detail"}
			applyUpdateRequest(c, tt.req)
			if !reflect.DeepEqual(*c, tt.expect) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, *c)
			}
		})
	}
}

func Test_IncFetchError_cov2(t *testing.T) {
	read := func() float64 {
		mfs, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			t.Fatalf("could not gather: %v", err)
		}
		for _, mf := range mfs {
			if mf.GetName() != "call_manager_outbound_config_fetch_error_total" {
				continue
			}
			for _, m := range mf.GetMetric() {
				for _, l := range m.GetLabel() {
					if l.GetName() == "error_type" && l.GetValue() == "cov2_test_type" {
						return m.GetCounter().GetValue()
					}
				}
			}
		}
		return 0
	}

	before := read()
	IncFetchError("cov2_test_type")
	IncFetchError("cov2_test_type")
	if after := read(); after != before+2 {
		t.Errorf("Wrong match. expect: %v, got: %v", before+2, after)
	}
}

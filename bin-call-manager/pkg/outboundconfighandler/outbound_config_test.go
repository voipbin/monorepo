package outboundconfighandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"monorepo/bin-call-manager/pkg/cachehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_outboundConfigHandler_GetByCustomerID(t *testing.T) {
	tests := []struct {
		name       string
		customerID uuid.UUID

		// cache expectations
		cacheGetResult *outboundconfig.OutboundConfig
		cacheGetErr    error

		// db expectations (only called on cache miss)
		dbGetResult *outboundconfig.OutboundConfig
		dbGetErr    error

		expectCacheSet        bool
		expectCacheSetNotFound bool
		expectRes             *outboundconfig.OutboundConfig
		expectErr             bool
	}{
		{
			name:           "cache hit - real config",
			customerID:     uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001"),
			cacheGetResult: &outboundconfig.OutboundConfig{Name: "cached"},
			cacheGetErr:    nil,
			expectRes:      &outboundconfig.OutboundConfig{Name: "cached"},
			expectErr:      false,
		},
		{
			name:           "negative cache hit - no DB call, return nil",
			customerID:     uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002"),
			cacheGetResult: nil,
			cacheGetErr:    nil,
			expectRes:      nil,
			expectErr:      false,
		},
		{
			name:                  "cache miss + DB found - write-through, return config",
			customerID:            uuid.FromStringOrNil("11111111-0000-0000-0000-000000000003"),
			cacheGetResult:        nil,
			cacheGetErr:           redis.Nil,
			dbGetResult:           &outboundconfig.OutboundConfig{Name: "from-db"},
			dbGetErr:              nil,
			expectCacheSet:        true,
			expectRes:             &outboundconfig.OutboundConfig{Name: "from-db"},
			expectErr:             false,
		},
		{
			name:                   "cache miss + DB not found - set sentinel, return nil",
			customerID:             uuid.FromStringOrNil("11111111-0000-0000-0000-000000000004"),
			cacheGetResult:         nil,
			cacheGetErr:            redis.Nil,
			dbGetResult:            nil,
			dbGetErr:               nil,
			expectCacheSetNotFound: true,
			expectRes:              nil,
			expectErr:              false,
		},
		{
			name:           "db error after cache miss - propagate error",
			customerID:     uuid.FromStringOrNil("11111111-0000-0000-0000-000000000005"),
			cacheGetResult: nil,
			cacheGetErr:    redis.Nil,
			dbGetResult:    nil,
			dbGetErr:       fmt.Errorf("db connection error"),
			expectRes:      nil,
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &outboundConfigHandler{
				utilHandler:  mockUtil,
				db:           mockDB,
				cacheHandler: mockCache,
			}

			ctx := context.Background()

			// Cache lookup always called
			mockCache.EXPECT().
				OutboundConfigGet(ctx, tt.customerID).
				Return(tt.cacheGetResult, tt.cacheGetErr).
				Times(1)

			// DB only called on cache miss (redis.Nil)
			if tt.cacheGetErr == redis.Nil {
				mockDB.EXPECT().
					OutboundConfigGetByCustomerID(ctx, tt.customerID).
					Return(tt.dbGetResult, tt.dbGetErr).
					Times(1)
			}

			// Cache write-through on DB hit
			if tt.expectCacheSet {
				mockCache.EXPECT().
					OutboundConfigSet(ctx, tt.customerID, tt.dbGetResult).
					Return(nil).
					Times(1)
			}

			// Negative sentinel set on DB miss
			if tt.expectCacheSetNotFound {
				mockCache.EXPECT().
					OutboundConfigSetNotFound(ctx, tt.customerID).
					Return(nil).
					Times(1)
			}

			got, err := h.GetByCustomerID(ctx, tt.customerID)
			if (err != nil) != tt.expectErr {
				t.Errorf("GetByCustomerID() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if tt.expectRes == nil && got != nil {
				t.Errorf("GetByCustomerID() = %v, want nil", got)
			}
			if tt.expectRes != nil {
				if got == nil {
					t.Errorf("GetByCustomerID() = nil, want %v", tt.expectRes)
					return
				}
				if got.Name != tt.expectRes.Name {
					t.Errorf("GetByCustomerID() Name = %v, want %v", got.Name, tt.expectRes.Name)
				}
			}
		})
	}
}

func Test_outboundConfigHandler_GetByID(t *testing.T) {
	tests := []struct {
		name   string
		id     uuid.UUID
		dbRes  *outboundconfig.OutboundConfig
		dbErr  error
		expect *outboundconfig.OutboundConfig
		wantErr bool
	}{
		{
			name:    "found",
			id:      uuid.FromStringOrNil("22222222-0000-0000-0000-000000000001"),
			dbRes:   &outboundconfig.OutboundConfig{Name: "cfg"},
			dbErr:   nil,
			expect:  &outboundconfig.OutboundConfig{Name: "cfg"},
			wantErr: false,
		},
		{
			name:    "not found",
			id:      uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002"),
			dbRes:   nil,
			dbErr:   nil,
			expect:  nil,
			wantErr: false,
		},
		{
			name:    "db error",
			id:      uuid.FromStringOrNil("22222222-0000-0000-0000-000000000003"),
			dbRes:   nil,
			dbErr:   fmt.Errorf("connection refused"),
			expect:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &outboundConfigHandler{
				utilHandler:  utilhandler.NewMockUtilHandler(mc),
				db:           mockDB,
				cacheHandler: cachehandler.NewMockCacheHandler(mc),
			}
			ctx := context.Background()

			mockDB.EXPECT().
				OutboundConfigGetByID(ctx, tt.id).
				Return(tt.dbRes, tt.dbErr).
				Times(1)

			got, err := h.GetByID(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.expect == nil && got != nil {
				t.Errorf("GetByID() = %v, want nil", got)
			}
			if tt.expect != nil && (got == nil || got.Name != tt.expect.Name) {
				t.Errorf("GetByID() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func Test_outboundConfigHandler_List(t *testing.T) {
	tests := []struct {
		name       string
		customerID uuid.UUID
		pageSize   uint64
		pageToken  string
		dbRes      []*outboundconfig.OutboundConfig
		dbErr      error
		wantCount  int
		wantErr    bool
	}{
		{
			name:       "returns list",
			customerID: uuid.FromStringOrNil("33333333-0000-0000-0000-000000000001"),
			pageSize:   10,
			pageToken:  "",
			dbRes: []*outboundconfig.OutboundConfig{
				{Name: "a"},
				{Name: "b"},
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:       "db error",
			customerID: uuid.FromStringOrNil("33333333-0000-0000-0000-000000000002"),
			pageSize:   10,
			dbRes:      nil,
			dbErr:      fmt.Errorf("db error"),
			wantCount:  0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &outboundConfigHandler{
				utilHandler:  utilhandler.NewMockUtilHandler(mc),
				db:           mockDB,
				cacheHandler: cachehandler.NewMockCacheHandler(mc),
			}
			ctx := context.Background()

			mockDB.EXPECT().
				OutboundConfigList(ctx, tt.customerID, tt.pageSize, tt.pageToken).
				Return(tt.dbRes, tt.dbErr).
				Times(1)

			got, err := h.List(ctx, tt.customerID, tt.pageSize, tt.pageToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("List() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(got) != tt.wantCount {
				t.Errorf("List() count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func Test_outboundConfigHandler_Create(t *testing.T) {
	name := "test-config"
	codecs := "PCMU,G729"
	wl := []string{"us", "gb"}

	tests := []struct {
		name       string
		customerID uuid.UUID
		req        *outboundconfig.UpdateRequest
		newID      uuid.UUID
		dbErr      error
		wantErr    bool
	}{
		{
			name:       "success",
			customerID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000001"),
			req: &outboundconfig.UpdateRequest{
				Name:                 &name,
				Codecs:               &codecs,
				DestinationWhitelist: &wl,
			},
			newID:   uuid.FromStringOrNil("44444444-0000-0000-0000-0000000000aa"),
			wantErr: false,
		},
		{
			name:       "db error",
			customerID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000002"),
			req:        &outboundconfig.UpdateRequest{Name: &name},
			newID:      uuid.FromStringOrNil("44444444-0000-0000-0000-0000000000bb"),
			dbErr:      fmt.Errorf("insert failed"),
			wantErr:    true,
		},
		{
			name:       "invalid codecs returns error without calling db",
			customerID: uuid.FromStringOrNil("44444444-0000-0000-0000-000000000003"),
			req: func() *outboundconfig.UpdateRequest {
				bad := "PCMU;G729" // semicolon not allowed
				return &outboundconfig.UpdateRequest{Codecs: &bad}
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &outboundConfigHandler{
				utilHandler:  mockUtil,
				db:           mockDB,
				cacheHandler: mockCache,
			}
			ctx := context.Background()

			// validation error path — no DB or UUID calls
			if tt.name == "invalid codecs returns error without calling db" {
				got, err := h.Create(ctx, tt.customerID, tt.req)
				if err == nil {
					t.Errorf("Create() expected error for invalid codecs, got nil (result=%v)", got)
				}
				return
			}

			mockUtil.EXPECT().UUIDCreate().Return(tt.newID).Times(1)
			mockDB.EXPECT().
				OutboundConfigCreate(ctx, gomock.Any()).
				Return(tt.dbErr).
				Times(1)

			if tt.dbErr == nil {
				mockCache.EXPECT().
					OutboundConfigSet(ctx, tt.customerID, gomock.Any()).
					Return(nil).
					Times(1)
			}

			got, err := h.Create(ctx, tt.customerID, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Errorf("Create() returned nil on success")
			}
			if !tt.wantErr && got != nil && got.ID != tt.newID {
				t.Errorf("Create() ID = %v, want %v", got.ID, tt.newID)
			}
		})
	}
}

func Test_outboundConfigHandler_Update(t *testing.T) {
	name := "updated"

	tests := []struct {
		name    string
		id      uuid.UUID
		req     *outboundconfig.UpdateRequest
		dbRes   *outboundconfig.OutboundConfig
		dbErr   error
		wantErr bool
	}{
		{
			name: "success - cache invalidated",
			id:   uuid.FromStringOrNil("55555555-0000-0000-0000-000000000001"),
			req:  &outboundconfig.UpdateRequest{Name: &name},
			dbRes: &outboundconfig.OutboundConfig{
				CustomerID: uuid.FromStringOrNil("55555555-0000-0000-0000-0000000000cc"),
				Name:       "updated",
			},
			wantErr: false,
		},
		{
			name:    "db error",
			id:      uuid.FromStringOrNil("55555555-0000-0000-0000-000000000002"),
			req:     &outboundconfig.UpdateRequest{Name: &name},
			dbRes:   nil,
			dbErr:   fmt.Errorf("update failed"),
			wantErr: true,
		},
		{
			name: "invalid whitelist - no db call",
			id:   uuid.FromStringOrNil("55555555-0000-0000-0000-000000000003"),
			req: func() *outboundconfig.UpdateRequest {
				bad := []string{"xx"} // invalid country code
				return &outboundconfig.UpdateRequest{DestinationWhitelist: &bad}
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := &outboundConfigHandler{
				utilHandler:  utilhandler.NewMockUtilHandler(mc),
				db:           mockDB,
				cacheHandler: mockCache,
			}
			ctx := context.Background()

			// validation error - no DB call
			if tt.name == "invalid whitelist - no db call" {
				got, err := h.Update(ctx, tt.id, tt.req)
				if err == nil {
					t.Errorf("Update() expected error for invalid whitelist, got nil (result=%v)", got)
				}
				return
			}

			mockDB.EXPECT().
				OutboundConfigUpdate(ctx, tt.id, tt.req).
				Return(tt.dbRes, tt.dbErr).
				Times(1)

			if tt.dbErr == nil && tt.dbRes != nil {
				mockCache.EXPECT().
					OutboundConfigDelete(ctx, tt.dbRes.CustomerID).
					Return(nil).
					Times(1)
			}

			got, err := h.Update(ctx, tt.id, tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil && tt.dbRes != nil {
				t.Errorf("Update() returned nil, want %v", tt.dbRes)
			}
		})
	}
}

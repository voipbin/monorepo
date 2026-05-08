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
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	nmnumber "monorepo/bin-number-manager/models/number"
)

func Test_outboundConfigHandler_Delete(t *testing.T) {
	tests := []struct {
		name string
		id   uuid.UUID

		dbGetRes *outboundconfig.OutboundConfig
		dbGetErr error

		dbDelErr error

		expectCacheDelete bool
		expectRes         *outboundconfig.OutboundConfig
		wantErr           bool
	}{
		{
			name: "success",
			id:   uuid.FromStringOrNil("66666666-0000-0000-0000-000000000001"),
			dbGetRes: &outboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("66666666-0000-0000-0000-000000000001"),
				CustomerID: uuid.FromStringOrNil("66666666-0000-0000-0000-0000000000cc"),
				Name:       "to-delete",
			},
			dbGetErr:          nil,
			dbDelErr:          nil,
			expectCacheDelete: true,
			expectRes:         &outboundconfig.OutboundConfig{Name: "to-delete"},
			wantErr:           false,
		},
		{
			name:              "GetByID error",
			id:                uuid.FromStringOrNil("66666666-0000-0000-0000-000000000002"),
			dbGetRes:          nil,
			dbGetErr:          fmt.Errorf("db connection error"),
			expectCacheDelete: false,
			expectRes:         nil,
			wantErr:           true,
		},
		{
			name:              "GetByID nil (not found)",
			id:                uuid.FromStringOrNil("66666666-0000-0000-0000-000000000003"),
			dbGetRes:          nil,
			dbGetErr:          nil,
			dbDelErr:          nil,
			expectCacheDelete: false,
			expectRes:         nil,
			wantErr:           false,
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
				reqHandler:   requesthandler.NewMockRequestHandler(mc),
			}
			ctx := context.Background()

			mockDB.EXPECT().
				OutboundConfigGetByID(ctx, tt.id).
				Return(tt.dbGetRes, tt.dbGetErr).
				Times(1)

			if tt.dbGetErr == nil {
				// OutboundConfigDelete is always called when GetByID returns no error
				mockDB.EXPECT().
					OutboundConfigDelete(ctx, tt.id).
					Return(tt.dbDelErr).
					Times(1)
			}

			if tt.expectCacheDelete {
				mockCache.EXPECT().
					OutboundConfigDelete(ctx, tt.dbGetRes.CustomerID).
					Return(nil).
					Times(1)
			}

			got, err := h.Delete(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectRes == nil && got != nil {
				t.Errorf("Delete() = %v, want nil", got)
				return
			}
			if tt.expectRes != nil {
				if got == nil {
					t.Errorf("Delete() = nil, want non-nil")
					return
				}
				if got.Name != tt.expectRes.Name {
					t.Errorf("Delete() Name = %v, want %v", got.Name, tt.expectRes.Name)
				}
				// After a successful delete, TMDelete must be set on the returned struct
				if tt.name == "success" && got.TMDelete == nil {
					t.Errorf("Delete() TMDelete is nil, want non-nil after delete")
				}
			}
		})
	}
}

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

		expectCacheSet         bool
		expectCacheSetNotFound bool
		expectRes              *outboundconfig.OutboundConfig
		expectErr              bool
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
			name:           "cache miss + DB found - write-through, return config",
			customerID:     uuid.FromStringOrNil("11111111-0000-0000-0000-000000000003"),
			cacheGetResult: nil,
			cacheGetErr:    redis.Nil,
			dbGetResult:    &outboundconfig.OutboundConfig{Name: "from-db"},
			dbGetErr:       nil,
			expectCacheSet: true,
			expectRes:      &outboundconfig.OutboundConfig{Name: "from-db"},
			expectErr:      false,
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
				reqHandler:   requesthandler.NewMockRequestHandler(mc),
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
		name    string
		id      uuid.UUID
		dbRes   *outboundconfig.OutboundConfig
		dbErr   error
		expect  *outboundconfig.OutboundConfig
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
				reqHandler:   requesthandler.NewMockRequestHandler(mc),
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
				reqHandler:   requesthandler.NewMockRequestHandler(mc),
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
				reqHandler:   requesthandler.NewMockRequestHandler(mc),
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

func Test_Create_WithDefaultOutgoingSourceNumberID(t *testing.T) {
	customerID := uuid.FromStringOrNil("eeeeeeee-0000-0000-0000-000000000001")
	defaultNumberID := uuid.FromStringOrNil("ffffffff-0000-0000-0000-000000000001")
	newID := uuid.FromStringOrNil("eeeeeeee-0000-0000-0000-0000000000aa")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &outboundConfigHandler{
		utilHandler:  mockUtil,
		db:           mockDB,
		cacheHandler: mockCache,
		reqHandler:   mockReq,
	}
	ctx := context.Background()

	req := &outboundconfig.UpdateRequest{
		DefaultOutgoingSourceNumberID: &defaultNumberID,
	}

	// validation step should look up the number
	expectFilters := map[nmnumber.Field]any{
		nmnumber.FieldCustomerID: customerID,
		nmnumber.FieldID:         defaultNumberID,
		nmnumber.FieldType:       nmnumber.TypeNormal,
		nmnumber.FieldStatus:     nmnumber.StatusActive,
		nmnumber.FieldDeleted:    false,
	}
	mockReq.EXPECT().
		NumberV1NumberList(ctx, "", uint64(1), expectFilters).
		Return([]nmnumber.Number{{Number: "+15551234567"}}, nil).
		Times(1)

	mockUtil.EXPECT().UUIDCreate().Return(newID).Times(1)

	// the OutboundConfig passed to OutboundConfigCreate must carry the field
	var captured *outboundconfig.OutboundConfig
	mockDB.EXPECT().
		OutboundConfigCreate(ctx, gomock.Any()).
		DoAndReturn(func(_ context.Context, c *outboundconfig.OutboundConfig) error {
			captured = c
			return nil
		}).
		Times(1)

	mockCache.EXPECT().OutboundConfigSet(ctx, customerID, gomock.Any()).Return(nil).Times(1)

	got, err := h.Create(ctx, customerID, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got == nil {
		t.Fatalf("Create() returned nil")
	}
	if got.DefaultOutgoingSourceNumberID != defaultNumberID {
		t.Errorf("Create() DefaultOutgoingSourceNumberID = %v, want %v", got.DefaultOutgoingSourceNumberID, defaultNumberID)
	}
	if captured == nil || captured.DefaultOutgoingSourceNumberID != defaultNumberID {
		t.Errorf("OutboundConfigCreate received DefaultOutgoingSourceNumberID = %v, want %v",
			captured.DefaultOutgoingSourceNumberID, defaultNumberID)
	}
}

func Test_outboundConfigHandler_Update(t *testing.T) {
	name := "updated"
	existingCustomerID := uuid.FromStringOrNil("55555555-0000-0000-0000-0000000000cc")

	tests := []struct {
		name string
		id   uuid.UUID
		req  *outboundconfig.UpdateRequest

		// return values for the GetByID call that Update now performs
		existingRes *outboundconfig.OutboundConfig
		existingErr error

		dbRes   *outboundconfig.OutboundConfig
		dbErr   error
		wantErr bool
	}{
		{
			name: "success - cache invalidated",
			id:   uuid.FromStringOrNil("55555555-0000-0000-0000-000000000001"),
			req:  &outboundconfig.UpdateRequest{Name: &name},
			existingRes: &outboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("55555555-0000-0000-0000-000000000001"),
				CustomerID: existingCustomerID,
			},
			dbRes: &outboundconfig.OutboundConfig{
				CustomerID: existingCustomerID,
				Name:       "updated",
			},
			wantErr: false,
		},
		{
			name: "db error",
			id:   uuid.FromStringOrNil("55555555-0000-0000-0000-000000000002"),
			req:  &outboundconfig.UpdateRequest{Name: &name},
			existingRes: &outboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("55555555-0000-0000-0000-000000000002"),
				CustomerID: existingCustomerID,
			},
			dbRes:   nil,
			dbErr:   fmt.Errorf("update failed"),
			wantErr: true,
		},
		{
			name: "invalid whitelist - no db update call",
			id:   uuid.FromStringOrNil("55555555-0000-0000-0000-000000000003"),
			req: func() *outboundconfig.UpdateRequest {
				bad := []string{"xx"} // invalid country code
				return &outboundconfig.UpdateRequest{DestinationWhitelist: &bad}
			}(),
			existingRes: &outboundconfig.OutboundConfig{
				ID:         uuid.FromStringOrNil("55555555-0000-0000-0000-000000000003"),
				CustomerID: existingCustomerID,
			},
			wantErr: true,
		},
		{
			name: "outbound_config not found",
			id:   uuid.FromStringOrNil("55555555-0000-0000-0000-000000000004"),
			req:  &outboundconfig.UpdateRequest{Name: &name},
			existingRes: nil,
			existingErr: nil,
			wantErr: true,
		},
		{
			name: "GetByID error",
			id:   uuid.FromStringOrNil("55555555-0000-0000-0000-000000000005"),
			req:  &outboundconfig.UpdateRequest{Name: &name},
			existingRes: nil,
			existingErr: fmt.Errorf("connection refused"),
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
				reqHandler:   requesthandler.NewMockRequestHandler(mc),
			}
			ctx := context.Background()

			// Update always calls GetByID first
			mockDB.EXPECT().
				OutboundConfigGetByID(ctx, tt.id).
				Return(tt.existingRes, tt.existingErr).
				Times(1)

			// validation error and lookup-error paths skip OutboundConfigUpdate
			if tt.existingErr == nil && tt.existingRes != nil && tt.name != "invalid whitelist - no db update call" {
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

func Test_Update_WithDefaultOutgoingSourceNumberID(t *testing.T) {
	id := uuid.FromStringOrNil("11221122-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("11221122-0000-0000-0000-0000000000cc")
	defaultNumberID := uuid.FromStringOrNil("11221122-0000-0000-0000-0000000000dd")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &outboundConfigHandler{
		utilHandler:  utilhandler.NewMockUtilHandler(mc),
		db:           mockDB,
		cacheHandler: mockCache,
		reqHandler:   mockReq,
	}
	ctx := context.Background()

	existing := &outboundconfig.OutboundConfig{
		ID:         id,
		CustomerID: customerID,
	}
	req := &outboundconfig.UpdateRequest{
		DefaultOutgoingSourceNumberID: &defaultNumberID,
	}

	// 1. GetByID for the customer lookup
	mockDB.EXPECT().OutboundConfigGetByID(ctx, id).Return(existing, nil).Times(1)

	// 2. validation: NumberV1NumberList must use the existing record's customer_id
	expectFilters := map[nmnumber.Field]any{
		nmnumber.FieldCustomerID: customerID,
		nmnumber.FieldID:         defaultNumberID,
		nmnumber.FieldType:       nmnumber.TypeNormal,
		nmnumber.FieldStatus:     nmnumber.StatusActive,
		nmnumber.FieldDeleted:    false,
	}
	mockReq.EXPECT().
		NumberV1NumberList(ctx, "", uint64(1), expectFilters).
		Return([]nmnumber.Number{{Number: "+15551234567"}}, nil).
		Times(1)

	// 3. db update + cache invalidation
	updated := &outboundconfig.OutboundConfig{
		ID:                            id,
		CustomerID:                    customerID,
		DefaultOutgoingSourceNumberID: defaultNumberID,
	}
	mockDB.EXPECT().OutboundConfigUpdate(ctx, id, req).Return(updated, nil).Times(1)
	mockCache.EXPECT().OutboundConfigDelete(ctx, customerID).Return(nil).Times(1)

	got, err := h.Update(ctx, id, req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got == nil {
		t.Fatalf("Update() returned nil")
	}
	if got.DefaultOutgoingSourceNumberID != defaultNumberID {
		t.Errorf("Update() DefaultOutgoingSourceNumberID = %v, want %v",
			got.DefaultOutgoingSourceNumberID, defaultNumberID)
	}
}

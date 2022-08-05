package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
)

func Test_ConferencecallCreate(t *testing.T) {

	tests := []struct {
		name string

		conferencecall *conferencecall.Conferencecall
		expectRes      *conferencecall.Conferencecall
	}{
		{
			"type call",

			&conferencecall.Conferencecall{
				ID:           uuid.FromStringOrNil("aab1d10e-2bb2-41fe-b519-273af50e8774"),
				CustomerID:   uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				ConferenceID: uuid.FromStringOrNil("edce43fd-8e5d-4178-b2e6-93479fa4f024"),

				ReferenceType: conferencecall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("27b3bfee-b030-404b-bca4-c30889ebb666"),
			},
			&conferencecall.Conferencecall{
				ID:           uuid.FromStringOrNil("aab1d10e-2bb2-41fe-b519-273af50e8774"),
				CustomerID:   uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				ConferenceID: uuid.FromStringOrNil("edce43fd-8e5d-4178-b2e6-93479fa4f024"),

				ReferenceType: conferencecall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("27b3bfee-b030-404b-bca4-c30889ebb666"),
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

			mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
			if err := h.ConferencecallCreate(ctx, tt.conferencecall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferencecallGet(ctx, tt.conferencecall.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
			res, err := h.ConferencecallGet(ctx, tt.conferencecall.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferencecallGetByReferenceID(t *testing.T) {

	type test struct {
		name string

		conferencecall *conferencecall.Conferencecall
	}

	tests := []test{
		{
			"normal",
			&conferencecall.Conferencecall{
				ID:            uuid.FromStringOrNil("1b4e752c-b766-4419-b778-73308e9607be"),
				CustomerID:    uuid.FromStringOrNil("1afc3ce2-9861-11ec-90b1-d76e949c3805"),
				ReferenceType: conferencecall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("371ebbd2-b52c-4e03-9444-0110f2b695cb"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
			if err := h.ConferencecallCreate(ctx, tt.conferencecall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.ConferencecallGetByReferenceID(ctx, tt.conferencecall.ReferenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.conferencecall, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.conferencecall, res)
			}
		})
	}
}

func Test_ConferencecallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		count      int
	}{
		{
			"normal",
			uuid.FromStringOrNil("8512e56c-cb08-46fa-96de-7855d0889577"),
			10,
		},
		{
			"empty",
			uuid.FromStringOrNil("9c61ef24-b396-465b-9705-44b420f2dc5d"),
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for i := 0; i < tt.count; i++ {
				cc := &conferencecall.Conferencecall{
					ID:         uuid.Must(uuid.NewV4()),
					CustomerID: tt.customerID,
					TMDelete:   DefaultTimeStamp,
				}

				mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
				_ = h.ConferencecallCreate(ctx, cc)
			}

			res, err := h.ConferencecallGetsByCustomerID(ctx, tt.customerID, 10, GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.count {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.count, len(res))
			}
		})
	}
}

func Test_ConferencecallGetsByConferenceID(t *testing.T) {

	tests := []struct {
		name string

		conferenceID uuid.UUID
		count        int
	}{
		{
			"normal",
			uuid.FromStringOrNil("e8871151-a0bb-4064-95a9-5b59b195ba96"),
			10,
		},
		{
			"empty",
			uuid.FromStringOrNil("2c6f5c63-293f-40bb-9c6e-12d15e3eca7b"),
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			for i := 0; i < tt.count; i++ {
				cc := &conferencecall.Conferencecall{
					ID:           uuid.Must(uuid.NewV4()),
					ConferenceID: tt.conferenceID,
					TMDelete:     DefaultTimeStamp,
				}

				mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
				_ = h.ConferencecallCreate(ctx, cc)
			}

			res, err := h.ConferencecallGetsByConferenceID(ctx, tt.conferenceID, 10, GetCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.count {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.count, len(res))
			}
		})
	}
}

func Test_ConferencecallDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID
	}{
		{
			"normal",
			uuid.FromStringOrNil("6304a458-b15e-4ea9-a1b8-43638e2df2a7"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			cc := &conferencecall.Conferencecall{
				ID: tt.id,
			}

			mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
			if err := h.ConferencecallCreate(ctx, cc); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
			if errDel := h.ConferencecallDelete(ctx, tt.id); errDel != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errDel)
			}

			mockCache.EXPECT().ConferencecallGet(ctx, tt.id).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
			res, err := h.ConferencecallGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete > GetCurTime() {
				t.Errorf("Wrong match. expect: small, got: %s", res.TMDelete)
			}

		})
	}
}

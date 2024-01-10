package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/cachehandler"
)

func Test_ConferencecallCreate(t *testing.T) {

	tests := []struct {
		name string

		conferencecall *conferencecall.Conferencecall

		responseCurTime string
		expectRes       *conferencecall.Conferencecall
	}{
		{
			"have all",

			&conferencecall.Conferencecall{
				ID:           uuid.FromStringOrNil("aab1d10e-2bb2-41fe-b519-273af50e8774"),
				CustomerID:   uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				ConferenceID: uuid.FromStringOrNil("edce43fd-8e5d-4178-b2e6-93479fa4f024"),

				ReferenceType: conferencecall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("27b3bfee-b030-404b-bca4-c30889ebb666"),

				Status: conferencecall.StatusJoining,
			},

			"2023-01-03 21:35:02.809",
			&conferencecall.Conferencecall{
				ID:           uuid.FromStringOrNil("aab1d10e-2bb2-41fe-b519-273af50e8774"),
				CustomerID:   uuid.FromStringOrNil("361de3de-7f45-11ec-b641-5358ec38b5e2"),
				ConferenceID: uuid.FromStringOrNil("edce43fd-8e5d-4178-b2e6-93479fa4f024"),

				ReferenceType: conferencecall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("27b3bfee-b030-404b-bca4-c30889ebb666"),

				Status: conferencecall.StatusJoining,

				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: DefaultTimeStamp,
				TMDelete: DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferencecallGetByReferenceID(t *testing.T) {

	type test struct {
		name           string
		conferencecall *conferencecall.Conferencecall

		referenceID uuid.UUID

		responseCurTime string
		expectRes       *conferencecall.Conferencecall
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

			uuid.FromStringOrNil("371ebbd2-b52c-4e03-9444-0110f2b695cb"),

			"2023-01-03 21:35:02.809",
			&conferencecall.Conferencecall{
				ID:            uuid.FromStringOrNil("1b4e752c-b766-4419-b778-73308e9607be"),
				CustomerID:    uuid.FromStringOrNil("1afc3ce2-9861-11ec-90b1-d76e949c3805"),
				ReferenceType: conferencecall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("371ebbd2-b52c-4e03-9444-0110f2b695cb"),
				TMCreate:      "2023-01-03 21:35:02.809",
				TMUpdate:      DefaultTimeStamp,
				TMDelete:      DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)

			h := &handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
			if err := h.ConferencecallCreate(ctx, tt.conferencecall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().ConferencecallGetByReferenceID(ctx, tt.referenceID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
			res, err := h.ConferencecallGetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferencecallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name            string
		conferencecalls []*conferencecall.Conferencecall

		customerID uuid.UUID
		count      int
		filters    map[string]string

		responseCurTime string

		expectRes []*conferencecall.Conferencecall
	}{
		{
			"normal",
			[]*conferencecall.Conferencecall{
				{
					ID:         uuid.FromStringOrNil("43b02684-94ce-11ed-95e6-3727def0e4fd"),
					CustomerID: uuid.FromStringOrNil("8512e56c-cb08-46fa-96de-7855d0889577"),
				},
				{
					ID:         uuid.FromStringOrNil("50be6c64-94ce-11ed-9def-dfcdc44a112d"),
					CustomerID: uuid.FromStringOrNil("8512e56c-cb08-46fa-96de-7855d0889577"),
				},
			},

			uuid.FromStringOrNil("8512e56c-cb08-46fa-96de-7855d0889577"),
			10,
			map[string]string{
				"deleted": "false",
			},

			"2023-01-03 21:35:02.809",
			[]*conferencecall.Conferencecall{
				{
					ID:         uuid.FromStringOrNil("43b02684-94ce-11ed-95e6-3727def0e4fd"),
					CustomerID: uuid.FromStringOrNil("8512e56c-cb08-46fa-96de-7855d0889577"),
					TMCreate:   "2023-01-03 21:35:02.809",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
				{
					ID:         uuid.FromStringOrNil("50be6c64-94ce-11ed-9def-dfcdc44a112d"),
					CustomerID: uuid.FromStringOrNil("8512e56c-cb08-46fa-96de-7855d0889577"),
					TMCreate:   "2023-01-03 21:35:02.809",
					TMUpdate:   DefaultTimeStamp,
					TMDelete:   DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*conferencecall.Conferencecall{},
			uuid.FromStringOrNil("9c61ef24-b396-465b-9705-44b420f2dc5d"),
			0,
			map[string]string{
				"deleted": "false",
			},

			"2023-01-03 21:35:02.809",
			[]*conferencecall.Conferencecall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			for _, cc := range tt.conferencecalls {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
				if errCreate := h.ConferencecallCreate(ctx, cc); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.ConferencecallGetsByCustomerID(ctx, tt.customerID, uint64(tt.count), utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferencecallGetsByConferenceID(t *testing.T) {

	tests := []struct {
		name            string
		conferencecalls []*conferencecall.Conferencecall

		conferenceID uuid.UUID
		count        int
		filters      map[string]string

		responseCurTime string
		expectRes       []*conferencecall.Conferencecall
	}{
		{
			"normal",
			[]*conferencecall.Conferencecall{
				{
					ID:           uuid.FromStringOrNil("79890bee-94cf-11ed-949c-4f9d80981a9d"),
					ConferenceID: uuid.FromStringOrNil("e8871151-a0bb-4064-95a9-5b59b195ba96"),
				},
				{
					ID:           uuid.FromStringOrNil("7a5b1134-94cf-11ed-a141-13e5119f35b2"),
					ConferenceID: uuid.FromStringOrNil("e8871151-a0bb-4064-95a9-5b59b195ba96"),
				},
				{
					ID:           uuid.FromStringOrNil("7a8365c6-94cf-11ed-968d-8b044e00c228"),
					ConferenceID: uuid.FromStringOrNil("7aa8bb82-94cf-11ed-a61f-9fdc5e434d99"),
				},
			},

			uuid.FromStringOrNil("e8871151-a0bb-4064-95a9-5b59b195ba96"),
			10,
			map[string]string{
				"deleted": "false",
			},

			"2023-01-03 21:35:02.809",
			[]*conferencecall.Conferencecall{
				{
					ID:           uuid.FromStringOrNil("79890bee-94cf-11ed-949c-4f9d80981a9d"),
					ConferenceID: uuid.FromStringOrNil("e8871151-a0bb-4064-95a9-5b59b195ba96"),
					TMCreate:     "2023-01-03 21:35:02.809",
					TMUpdate:     DefaultTimeStamp,
					TMDelete:     DefaultTimeStamp,
				},
				{
					ID:           uuid.FromStringOrNil("7a5b1134-94cf-11ed-a141-13e5119f35b2"),
					ConferenceID: uuid.FromStringOrNil("e8871151-a0bb-4064-95a9-5b59b195ba96"),
					TMCreate:     "2023-01-03 21:35:02.809",
					TMUpdate:     DefaultTimeStamp,
					TMDelete:     DefaultTimeStamp,
				},
			},
		},
		{
			"empty",
			[]*conferencecall.Conferencecall{},

			uuid.FromStringOrNil("2c6f5c63-293f-40bb-9c6e-12d15e3eca7b"),
			0,
			map[string]string{
				"deleted": "false",
			},

			"2023-01-03 21:35:02.809",
			[]*conferencecall.Conferencecall{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			for _, cc := range tt.conferencecalls {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
				if errCreate := h.ConferencecallCreate(ctx, cc); errCreate != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errCreate)
				}
			}

			res, err := h.ConferencecallGetsByConferenceID(ctx, tt.conferenceID, uint64(tt.count), utilhandler.TimeGetCurTime(), tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ConferencecallDelete(t *testing.T) {

	tests := []struct {
		name           string
		conferencecall *conferencecall.Conferencecall

		id uuid.UUID

		responseCurTime string
		expectRes       *conferencecall.Conferencecall
	}{
		{
			"normal",
			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("284a800e-94d0-11ed-89ad-c749aec641a6"),
			},

			uuid.FromStringOrNil("284a800e-94d0-11ed-89ad-c749aec641a6"),

			"2023-01-03 21:35:02.809",
			&conferencecall.Conferencecall{
				ID:       uuid.FromStringOrNil("284a800e-94d0-11ed-89ad-c749aec641a6"),
				TMCreate: "2023-01-03 21:35:02.809",
				TMUpdate: "2023-01-03 21:35:02.809",
				TMDelete: "2023-01-03 21:35:02.809",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().ConferencecallSet(ctx, gomock.Any())
			if err := h.ConferencecallCreate(ctx, tt.conferencecall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
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

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/message-manager.git/models/target"
	"gitlab.com/voipbin/bin-manager/message-manager.git/pkg/cachehandler"
)

func Test_MessageCreate(t *testing.T) {
	tests := []struct {
		name            string
		message         *message.Message
		responseCurTime string
		expectRes       *message.Message
	}{
		{
			"test normal",
			&message.Message{
				ID:         uuid.FromStringOrNil("f5f2cefa-a055-11ec-a0d1-c7b28923b1f5"),
				CustomerID: uuid.FromStringOrNil("326ef638-a056-11ec-95de-6b924aa3ef53"),
				Type:       message.TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+821100000002",
						},
						Status: target.StatusSent,
					},
				},
				ProviderName:        message.ProviderNameMessagebird,
				ProviderReferenceID: "6b79e50e426c4d64ac45345bae84fe55",
				Text:                "Hello, this is test message.",
				Medias:              []string{},
				Direction:           message.DirectionOutbound,
			},

			"2021-02-26 18:26:49.000",
			&message.Message{
				ID:         uuid.FromStringOrNil("f5f2cefa-a055-11ec-a0d1-c7b28923b1f5"),
				CustomerID: uuid.FromStringOrNil("326ef638-a056-11ec-95de-6b924aa3ef53"),
				Type:       message.TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+821100000002",
						},
						Status: target.StatusSent,
					},
				},
				ProviderName:        message.ProviderNameMessagebird,
				ProviderReferenceID: "6b79e50e426c4d64ac45345bae84fe55",
				Text:                "Hello, this is test message.",
				Medias:              []string{},
				Direction:           message.DirectionOutbound,
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            DefaultTimeStamp,
				TMDelete:            DefaultTimeStamp,
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
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().MessageGet(ctx, tt.message.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
			res, err := h.MessageGet(ctx, tt.message.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_MessageDelete(t *testing.T) {

	tests := []struct {
		name    string
		message *message.Message

		responseCurTime string
		expectRes       *message.Message
	}{
		{
			"test normal",
			&message.Message{
				ID:         uuid.FromStringOrNil("fc67b82c-a2a3-11ec-970f-1f9f06c64b70"),
				CustomerID: uuid.FromStringOrNil("3f7a4c24-a2a4-11ec-b26e-3f8d47c2b450"),
				Type:       message.TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+821100000002",
						},
						Status: target.StatusSent,
					},
				},
				ProviderName:        message.ProviderNameMessagebird,
				ProviderReferenceID: "6b79e50e426c4d64ac45345bae84fe55",
				Text:                "Hello, this is test message.",
				Medias:              []string{},
				Direction:           message.DirectionOutbound,
			},

			"2021-02-26 18:26:49.000",
			&message.Message{
				ID:         uuid.FromStringOrNil("fc67b82c-a2a3-11ec-970f-1f9f06c64b70"),
				CustomerID: uuid.FromStringOrNil("3f7a4c24-a2a4-11ec-b26e-3f8d47c2b450"),
				Type:       message.TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+821100000002",
						},
						Status: target.StatusSent,
					},
				},
				ProviderName:        message.ProviderNameMessagebird,
				ProviderReferenceID: "6b79e50e426c4d64ac45345bae84fe55",
				Text:                "Hello, this is test message.",
				Medias:              []string{},
				Direction:           message.DirectionOutbound,
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            "2021-02-26 18:26:49.000",
				TMDelete:            "2021-02-26 18:26:49.000",
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
				cache:       mockCache,
				db:          dbTest,
				utilHandler: mockUtil,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageDelete(ctx, tt.message.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().MessageGet(ctx, tt.message.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
			res, err := h.MessageGet(ctx, tt.message.ID)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_MessageUpdateTargets(t *testing.T) {

	tests := []struct {
		name    string
		message *message.Message

		targets []target.Target

		responseCurTime string
		expectRes       *message.Message
	}{
		{
			"test normal",
			&message.Message{
				ID:         uuid.FromStringOrNil("4757235a-a226-11ec-9834-f70b08e3860f"),
				CustomerID: uuid.FromStringOrNil("502469b6-a226-11ec-aedf-9fd7c533e572"),
				Type:       message.TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+821100000002",
						},
						Status: target.StatusQueued,
					},
				},
				ProviderName:        message.ProviderNameMessagebird,
				ProviderReferenceID: "6b79e50e426c4d64ac45345bae84fe55",
				Text:                "Hello, this is test message.",
				Medias:              []string{},
				Direction:           message.DirectionOutbound,
			},

			[]target.Target{
				{
					Destination: commonaddress.Address{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
					Status: target.StatusSent,
				},
			},

			"2021-02-26 18:26:49.000",
			&message.Message{
				ID:         uuid.FromStringOrNil("4757235a-a226-11ec-9834-f70b08e3860f"),
				CustomerID: uuid.FromStringOrNil("502469b6-a226-11ec-aedf-9fd7c533e572"),
				Type:       message.TypeSMS,
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Targets: []target.Target{
					{
						Destination: commonaddress.Address{
							Type:   commonaddress.TypeTel,
							Target: "+821100000002",
						},
						Status: target.StatusSent,
					},
				},
				ProviderName:        message.ProviderNameMessagebird,
				ProviderReferenceID: "6b79e50e426c4d64ac45345bae84fe55",
				Text:                "Hello, this is test message.",
				Medias:              []string{},
				Direction:           message.DirectionOutbound,
				TMCreate:            "2021-02-26 18:26:49.000",
				TMUpdate:            "2021-02-26 18:26:49.000",
				TMDelete:            DefaultTimeStamp,
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
			mockCache.EXPECT().MessageSet(ctx, gomock.Any())
			if err := h.MessageCreate(ctx, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
			if errTargets := h.MessageUpdateTargets(ctx, tt.message.ID, tt.targets); errTargets != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errTargets)
			}

			mockCache.EXPECT().MessageGet(ctx, tt.message.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().MessageSet(ctx, gomock.Any()).Return(nil)
			res, err := h.MessageGet(ctx, tt.message.ID)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_MessageGets(t *testing.T) {

	tests := []struct {
		name     string
		messages []*message.Message

		customerID uuid.UUID

		responseCurTime string
		expectCount     int
	}{
		{
			"normal",
			[]*message.Message{
				{
					ID:         uuid.FromStringOrNil("a7dbbd7e-a296-11ec-b88e-07268af4c3b0"),
					CustomerID: uuid.FromStringOrNil("a73a34f4-a296-11ec-b7df-a3ed77d36f0d"),

					TMCreate: "2021-01-01 00:00:00.000",
					TMUpdate: DefaultTimeStamp,
					TMDelete: DefaultTimeStamp,
				},
			},

			uuid.FromStringOrNil("a73a34f4-a296-11ec-b7df-a3ed77d36f0d"),

			"2021-01-01 00:00:00.000",
			1,
		},
		{
			"empty",
			[]*message.Message{},

			uuid.FromStringOrNil("a8053398-a296-11ec-a7c7-33a89a071234"),

			"",
			0,
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

			// creates messages for test
			for i := 0; i < len(tt.messages); i++ {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().MessageSet(ctx, gomock.Any())

				if err := h.MessageCreate(ctx, tt.messages[i]); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.MessageGets(ctx, tt.customerID, 10, utilhandler.TimeGetCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.expectCount {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expectCount, len(res))
			}
		})
	}
}

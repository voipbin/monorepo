package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	uuid "github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	_ "github.com/mattn/go-sqlite3"

	"gitlab.com/voipbin/bin-manager/number-manager.git/models"
	"gitlab.com/voipbin/bin-manager/number-manager.git/pkg/cachehandler"
)

func TestNumberCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name         string
		number       *models.Number
		expectNumber *models.Number
	}

	tests := []test{
		{
			"test normal",
			&models.Number{
				ID:                  uuid.FromStringOrNil("8290e0be-7905-11eb-90c7-d3d5addc947a"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              models.NumberStatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
			&models.Number{
				ID:                  uuid.FromStringOrNil("8290e0be-7905-11eb-90c7-d3d5addc947a"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              models.NumberStatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberCreate(context.Background(), tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			res, err := h.NumberGet(context.Background(), tt.number.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMCreate = ""
			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectNumber, res)
			}
		})
	}
}

func TestNumberGets(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		userID uint64
		minNum int
	}

	tests := []test{
		{
			"normal",
			1,
			1,
		},
		{
			"empty",
			2,
			0,
		},
	}

	// creates numbers for test
	h := NewHandler(dbTest, mockCache)
	mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
	mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
	h.NumberCreate(context.Background(), &models.Number{ID: uuid.FromStringOrNil("82337ace-790e-11eb-a269-f75aee0055a8"), UserID: 1})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := h.NumberGets(context.Background(), tt.userID, 10, getCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) < tt.minNum {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.minNum, len(res))
			}
		})
	}
}

func TestNumberGetsByFlowID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name string

		flowID  uuid.UUID
		numbers []*models.Number

		expectNum int
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("66beabfe-7d20-11eb-9b69-375c485b40fa"),
			[]*models.Number{
				{
					ID:     uuid.FromStringOrNil("5d73b940-7d20-11eb-8335-97856a00f2c6"),
					UserID: 1,
					FlowID: uuid.FromStringOrNil("66beabfe-7d20-11eb-9b69-375c485b40fa"),
				},
			},
			1,
		},
		{
			"3 flows, but grep 2",
			uuid.FromStringOrNil("0472a166-7d21-11eb-ab7a-93bacc9ce3f2"),
			[]*models.Number{
				{
					ID:     uuid.FromStringOrNil("109347b6-7d21-11eb-bdd4-c7226a0e1c81"),
					UserID: 1,
					FlowID: uuid.FromStringOrNil("0472a166-7d21-11eb-ab7a-93bacc9ce3f2"),
				},
				{
					ID:     uuid.FromStringOrNil("10b60706-7d21-11eb-90ae-2305526adf47"),
					UserID: 1,
					FlowID: uuid.FromStringOrNil("0472a166-7d21-11eb-ab7a-93bacc9ce3f2"),
				},
				{
					ID:     uuid.FromStringOrNil("10cf5ee0-7d21-11eb-9733-b73b63288625"),
					UserID: 1,
					FlowID: uuid.FromStringOrNil("10eff100-7d21-11eb-b275-6ff5cde65beb"),
				},
			},
			2,
		},
	}

	// creates numbers for test
	h := NewHandler(dbTest, mockCache)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// create numbers
			for _, n := range tt.numbers {
				mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
				mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
				h.NumberCreate(ctx, n)
			}

			res, err := h.NumberGetsByFlowID(ctx, tt.flowID, 100, getCurTime())
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != tt.expectNum {
				t.Errorf("Wrong match. expect: %d, got: %v", tt.expectNum, len(res))
			}
		})
	}
}

func TestNumberDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name         string
		number       *models.Number
		expectNumber *models.Number
	}

	tests := []test{
		{
			"test normal",
			&models.Number{
				ID:                  uuid.FromStringOrNil("13218b0c-790f-11eb-9553-2f17a3e27acb"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              models.NumberStatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},

			&models.Number{
				ID:                  uuid.FromStringOrNil("13218b0c-790f-11eb-9553-2f17a3e27acb"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              models.NumberStatusDeleted,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberDelete(ctx, tt.number.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			res, err := h.NumberGet(context.Background(), tt.number.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == "" || res.TMUpdate == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}
			res.TMCreate = ""
			res.TMDelete = ""
			res.TMUpdate = ""

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectNumber, res)
			}
		})
	}
}

func TestNumberUpdate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockCache := cachehandler.NewMockCacheHandler(mc)

	type test struct {
		name         string
		number       *models.Number
		updateNumber *models.Number
		expectNumber *models.Number
	}

	tests := []test{
		{
			"test normal",
			&models.Number{
				ID:                  uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
				Number:              "+821021656521",
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              models.NumberStatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
			&models.Number{
				ID:     uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
				FlowID: uuid.FromStringOrNil("9496e31a-7c54-11eb-915d-3f8ab244a929"),
			},
			&models.Number{
				ID:                  uuid.FromStringOrNil("88df0e44-7c54-11eb-b2f8-37f9f70b06cd"),
				Number:              "+821021656521",
				FlowID:              uuid.FromStringOrNil("9496e31a-7c54-11eb-915d-3f8ab244a929"),
				UserID:              1,
				ProviderName:        "telnyx",
				ProviderReferenceID: "1580568175064384684",
				Status:              models.NumberStatusActive,
				T38Enabled:          true,
				EmergencyEnabled:    false,
				TMPurchase:          "2021-02-26 18:26:49.000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(dbTest, mockCache)
			ctx := context.Background()

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberCreate(ctx, tt.number); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			if err := h.NumberUpdate(ctx, tt.updateNumber); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().NumberGet(gomock.Any(), tt.number.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().NumberSet(gomock.Any(), gomock.Any())
			mockCache.EXPECT().NumberSetByNumber(gomock.Any(), gomock.Any())
			res, err := h.NumberGet(context.Background(), tt.number.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMUpdate == "" {
				t.Errorf("Wrong match. expect: not empty, got: empty")
			}
			res.TMCreate = ""
			res.TMDelete = ""
			res.TMUpdate = ""

			if reflect.DeepEqual(tt.expectNumber, res) == false {
				t.Errorf("Wrong match.\nexpect: %v,\ngot: %v\n", tt.expectNumber, res)
			}
		})
	}
}

// func TestCallSetStatus(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	type test struct {
// 		name     string
// 		id       uuid.UUID
// 		flowID   uuid.UUID
// 		status   call.Status
// 		tmUpdate string

// 		call       *call.Call
// 		expectCall call.Call
// 	}

// 	tests := []test{
// 		{
// 			"test normal",
// 			uuid.Must(uuid.NewV4()),
// 			uuid.Must(uuid.NewV4()),
// 			call.StatusProgressing,
// 			"2020-04-18T03:22:18.995000",
// 			&call.Call{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			call.Call{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,

// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusProgressing,
// 				Direction: call.DirectionIncoming,

// 				TMCreate:      "2020-04-18T03:22:17.995000",
// 				TMProgressing: "2020-04-18T03:22:18.995000",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			h := NewHandler(dbTest, mockCache)

// 			tt.call.ID = tt.id
// 			tt.call.FlowID = tt.flowID
// 			tt.expectCall.ID = tt.id
// 			tt.expectCall.FlowID = tt.flowID
// 			tt.expectCall.TMUpdate = tt.tmUpdate

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallCreate(context.Background(), tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallGet(gomock.Any(), tt.id).Return(tt.call, nil)
// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallSetStatus(context.Background(), tt.id, tt.status, tt.tmUpdate); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			res, err := h.CallGet(context.Background(), tt.call.ID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			tt.expectCall.TMUpdate = res.TMUpdate
// 			if reflect.DeepEqual(tt.expectCall, *res) == false {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
// 			}
// 		})
// 	}
// }

// func TestCallGetByChannelID(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	type test struct {
// 		name   string
// 		id     uuid.UUID
// 		flowID uuid.UUID

// 		call       call.Call
// 		expectCall call.Call
// 	}

// 	tests := []test{
// 		{
// 			"test normal",
// 			uuid.Must(uuid.NewV4()),
// 			uuid.Must(uuid.NewV4()),
// 			call.Call{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "2505d858-8687-11ea-8723-d35628256201",
// 				Type:       call.TypeFlow,

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			call.Call{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "2505d858-8687-11ea-8723-d35628256201",
// 				Type:       call.TypeFlow,

// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 		},
// 		{
// 			"test normal has source address type sip",
// 			uuid.Must(uuid.NewV4()),
// 			uuid.Must(uuid.NewV4()),
// 			call.Call{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "2aa510da-8687-11ea-b1b4-3f62cf9e4def",
// 				Type:       call.TypeFlow,

// 				Source: call.Address{
// 					Type: call.AddressTypeSIP,
// 				},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			call.Call{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "2aa510da-8687-11ea-b1b4-3f62cf9e4def",
// 				Type:       call.TypeFlow,

// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				Source: call.Address{
// 					Type: call.AddressTypeSIP,
// 				},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			h := NewHandler(dbTest, mockCache)

// 			tt.call.ID = tt.id
// 			tt.call.FlowID = tt.flowID
// 			tt.expectCall.ID = tt.id
// 			tt.expectCall.FlowID = tt.flowID

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallCreate(context.Background(), &tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res, err := h.CallGetByChannelID(context.Background(), tt.call.ChannelID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 			t.Logf("Created call. call: %v", res)

// 			if reflect.DeepEqual(tt.expectCall, *res) == false {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
// 			}
// 		})
// 	}
// }

// func TestCallCallSetHangup(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	type test struct {
// 		name     string
// 		id       uuid.UUID
// 		reason   call.HangupReason
// 		hangupBy call.HangupBy
// 		tmUpdate string

// 		call       *call.Call
// 		expectCall call.Call
// 	}

// 	tests := []test{
// 		{
// 			"test normal",
// 			uuid.Must(uuid.NewV4()),
// 			call.HangupReasonNormal,
// 			call.HangupByLocal,
// 			"2020-04-18T03:22:18.995000",
// 			&call.Call{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			call.Call{
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,

// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusHangup,
// 				Direction: call.DirectionIncoming,

// 				HangupReason: call.HangupReasonNormal,
// 				HangupBy:     call.HangupByLocal,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 				TMUpdate: "2020-04-18T03:22:18.995000",
// 				TMHangup: "2020-04-18T03:22:18.995000",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			h := NewHandler(dbTest, mockCache)

// 			tt.call.ID = tt.id
// 			tt.expectCall.ID = tt.id
// 			tt.expectCall.TMUpdate = tt.tmUpdate

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallCreate(context.Background(), tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallSetHangup(context.Background(), tt.id, tt.reason, tt.hangupBy, tt.tmUpdate); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			res, err := h.CallGet(context.Background(), tt.call.ID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if reflect.DeepEqual(tt.expectCall, *res) == false {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
// 			}
// 		})
// 	}
// }

// func TestCallSetFlowID(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	type test struct {
// 		name   string
// 		flowID uuid.UUID
// 		call   *call.Call

// 		expectCall *call.Call
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			uuid.FromStringOrNil("52f4a50a-8cc7-11ea-87f7-f36a8e4090eb"),
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("3599ce5e-9357-11ea-b215-f7ddc7ee506e"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("3599ce5e-9357-11ea-b215-f7ddc7ee506e"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,

// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				FlowID: uuid.FromStringOrNil("52f4a50a-8cc7-11ea-87f7-f36a8e4090eb"),

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			h := NewHandler(dbTest, mockCache)

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallCreate(context.Background(), tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallSetFlowID(context.Background(), tt.call.ID, tt.flowID); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			res, err := h.CallGet(context.Background(), tt.call.ID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res.TMUpdate = ""
// 			if reflect.DeepEqual(tt.expectCall, res) == false {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
// 			}
// 		})
// 	}
// }

// func TestCallSetConferenceID(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	type test struct {
// 		name         string
// 		conferenceID uuid.UUID
// 		call         *call.Call

// 		expectCall *call.Call
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			uuid.FromStringOrNil("62faff48-9358-11ea-8455-8fd1af79d7dc"),
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("56ca1f9c-9358-11ea-8dd7-472b84a9f7d4"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("56ca1f9c-9358-11ea-8dd7-472b84a9f7d4"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,

// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				ConfID: uuid.FromStringOrNil("62faff48-9358-11ea-8455-8fd1af79d7dc"),

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			h := NewHandler(dbTest, mockCache)

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallCreate(context.Background(), tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallSetConferenceID(context.Background(), tt.call.ID, tt.conferenceID); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			res, err := h.CallGet(context.Background(), tt.call.ID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res.TMUpdate = ""
// 			if reflect.DeepEqual(tt.expectCall, res) == false {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
// 			}
// 		})
// 	}
// }

// func TestCallSetAction(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	type test struct {
// 		name   string
// 		call   *call.Call
// 		action *action.Action

// 		expectCall *call.Call
// 	}

// 	tests := []test{
// 		{
// 			"echo option duration",
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("1d55d302-8d02-11ea-992f-53a0113a8a9b"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,
// 				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			&action.Action{
// 				ID:     uuid.FromStringOrNil("dc455d64-8d02-11ea-9d6e-0b6fe8f7bdc6"),
// 				Type:   action.TypeEcho,
// 				Option: []byte(`{"duration":180}`),
// 			},

// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("1d55d302-8d02-11ea-992f-53a0113a8a9b"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "93ea5e38-84e3-11ea-8927-dbf157fd2c9a",
// 				Type:       call.TypeFlow,
// 				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Action: action.Action{
// 					ID:     uuid.FromStringOrNil("dc455d64-8d02-11ea-9d6e-0b6fe8f7bdc6"),
// 					Type:   action.TypeEcho,
// 					Option: []byte(`{"duration":180}`),
// 				},
// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 		},

// 		{
// 			"echo option empty",
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("964b060e-8d04-11ea-bc42-93d5d0871556"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "9c5c8e5a-8d04-11ea-9e62-3be93b94e0eb",
// 				Type:       call.TypeFlow,
// 				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			&action.Action{
// 				ID:   uuid.FromStringOrNil("a1e3ff02-8d04-11ea-b30b-9fb57c4036f4"),
// 				Type: action.TypeEcho,
// 			},

// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("964b060e-8d04-11ea-bc42-93d5d0871556"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "9c5c8e5a-8d04-11ea-9e62-3be93b94e0eb",
// 				Type:       call.TypeFlow,
// 				FlowID:     uuid.FromStringOrNil("11dd8344-8d02-11ea-9aef-334a6a41cb02"),

// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				Source:      call.Address{},
// 				Destination: call.Address{},

// 				Action: action.Action{
// 					ID:   uuid.FromStringOrNil("a1e3ff02-8d04-11ea-b30b-9fb57c4036f4"),
// 					Type: action.TypeEcho,
// 				},
// 				Status:    call.StatusRinging,
// 				Direction: call.DirectionIncoming,

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			h := NewHandler(dbTest, mockCache)

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallCreate(context.Background(), tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallSetAction(context.Background(), tt.call.ID, tt.action); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			res, err := h.CallGet(context.Background(), tt.call.ID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res.TMUpdate = ""
// 			if reflect.DeepEqual(*tt.expectCall, *res) == false {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
// 			}
// 		})
// 	}
// }

// func TestCallSetMasterCallID(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	type test struct {
// 		name         string
// 		call         *call.Call
// 		masterCallID uuid.UUID

// 		expectCall *call.Call
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("14649d2c-24fc-11eb-bb0b-9bd6970f725f"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "14daba5c-24fc-11eb-8f58-8b798baaf553",
// 				Type:       call.TypeFlow,
// 				TMCreate:   "2020-04-18T03:22:17.995000",
// 			},
// 			uuid.FromStringOrNil("4a6ce0aa-24fc-11eb-aec0-4b97b9a2422a"),
// 			&call.Call{
// 				ID:             uuid.FromStringOrNil("14649d2c-24fc-11eb-bb0b-9bd6970f725f"),
// 				AsteriskID:     "3e:50:6b:43:bb:30",
// 				ChannelID:      "14daba5c-24fc-11eb-8f58-8b798baaf553",
// 				Type:           call.TypeFlow,
// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},
// 				MasterCallID:   uuid.FromStringOrNil("4a6ce0aa-24fc-11eb-aec0-4b97b9a2422a"),
// 				TMCreate:       "2020-04-18T03:22:17.995000",
// 			},
// 		},
// 		{
// 			"set nil",
// 			&call.Call{
// 				ID:       uuid.FromStringOrNil("665db8f2-2501-11eb-86ce-f3a50eef6f26"),
// 				Type:     call.TypeFlow,
// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			uuid.Nil,
// 			&call.Call{
// 				ID:             uuid.FromStringOrNil("665db8f2-2501-11eb-86ce-f3a50eef6f26"),
// 				Type:           call.TypeFlow,
// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},
// 				TMCreate:       "2020-04-18T03:22:17.995000",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			h := NewHandler(dbTest, mockCache)

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallCreate(context.Background(), tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallSetMasterCallID(context.Background(), tt.call.ID, tt.masterCallID); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			res, err := h.CallGet(context.Background(), tt.call.ID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res.TMUpdate = ""
// 			if reflect.DeepEqual(*tt.expectCall, *res) == false {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
// 			}
// 		})
// 	}
// }

// func TestCallSetRecordID(t *testing.T) {
// 	mc := gomock.NewController(t)
// 	defer mc.Finish()

// 	mockCache := cachehandler.NewMockCacheHandler(mc)

// 	type test struct {
// 		name     string
// 		call     *call.Call
// 		reocrdID uuid.UUID

// 		expectCall *call.Call
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			&call.Call{
// 				ID:         uuid.FromStringOrNil("46ab9ad8-282b-11eb-82c3-6782faf5e030"),
// 				AsteriskID: "3e:50:6b:43:bb:30",
// 				ChannelID:  "4e2fe520-282b-11eb-ad66-b777dce59261",
// 				Type:       call.TypeFlow,
// 				TMCreate:   "2020-04-18T03:22:17.995000",
// 			},
// 			uuid.FromStringOrNil("4e847572-282b-11eb-9c58-97622e4406e2"),
// 			&call.Call{
// 				ID:             uuid.FromStringOrNil("46ab9ad8-282b-11eb-82c3-6782faf5e030"),
// 				AsteriskID:     "3e:50:6b:43:bb:30",
// 				ChannelID:      "4e2fe520-282b-11eb-ad66-b777dce59261",
// 				Type:           call.TypeFlow,
// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				RecordingID: uuid.FromStringOrNil("4e847572-282b-11eb-9c58-97622e4406e2"),
// 				TMCreate:    "2020-04-18T03:22:17.995000",
// 			},
// 		},
// 		{
// 			"set empty",
// 			&call.Call{
// 				ID:       uuid.FromStringOrNil("7b3e197e-282b-11eb-956d-4feb054947db"),
// 				Type:     call.TypeFlow,
// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 			uuid.Nil,
// 			&call.Call{
// 				ID:             uuid.FromStringOrNil("7b3e197e-282b-11eb-956d-4feb054947db"),
// 				Type:           call.TypeFlow,
// 				ChainedCallIDs: []uuid.UUID{},
// 				RecordingIDs:   []uuid.UUID{},

// 				TMCreate: "2020-04-18T03:22:17.995000",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			h := NewHandler(dbTest, mockCache)

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallCreate(context.Background(), tt.call); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			if err := h.CallSetRecordID(context.Background(), tt.call.ID, tt.reocrdID); err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			mockCache.EXPECT().CallGet(gomock.Any(), tt.call.ID).Return(nil, fmt.Errorf(""))
// 			mockCache.EXPECT().CallSet(gomock.Any(), gomock.Any())
// 			res, err := h.CallGet(context.Background(), tt.call.ID)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			res.TMUpdate = ""
// 			if reflect.DeepEqual(*tt.expectCall, *res) == false {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectCall, res)
// 			}
// 		})
// 	}
// }

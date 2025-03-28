package summaryhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		activeflowID  uuid.UUID
		referenceType summary.ReferenceType
		referenceID   uuid.UUID
		status        summary.Status
		language      string
		content       string

		responseUUID uuid.UUID

		expectedSummary   *summary.Summary
		expectedVariables map[string]string
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("f227397c-f260-11ef-b217-4f6ff6930cf2"),
			activeflowID:  uuid.FromStringOrNil("581fc4fa-0b8f-11f0-9c7f-0bb793ad8854"),
			referenceType: summary.ReferenceTypeRecording,
			referenceID:   uuid.FromStringOrNil("578e0f60-0b8f-11f0-928e-a318d8221ca5"),
			status:        summary.StatusDone,
			language:      "en-US",
			content:       "Hello, world!",

			responseUUID: uuid.FromStringOrNil("57c44d50-0b8f-11f0-91ab-174598f05899"),

			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("57c44d50-0b8f-11f0-91ab-174598f05899"),
					CustomerID: uuid.FromStringOrNil("f227397c-f260-11ef-b217-4f6ff6930cf2"),
				},

				ActiveflowID:  uuid.FromStringOrNil("581fc4fa-0b8f-11f0-9c7f-0bb793ad8854"),
				ReferenceType: summary.ReferenceTypeRecording,
				ReferenceID:   uuid.FromStringOrNil("578e0f60-0b8f-11f0-928e-a318d8221ca5"),

				Status:   summary.StatusDone,
				Language: "en-US",
				Content:  "Hello, world!",
			},
			expectedVariables: map[string]string{
				variableSummaryID:            "57c44d50-0b8f-11f0-91ab-174598f05899",
				variableSummaryReferenceType: string(summary.ReferenceTypeRecording),
				variableSummaryReferenceID:   "578e0f60-0b8f-11f0-928e-a318d8221ca5",
				variableSummaryLanguage:      "en-US",
				variableSummaryContent:       "Hello, world!",
			},
		},
		{
			name: "empty",

			responseUUID: uuid.FromStringOrNil("57f1ce10-0b8f-11f0-8383-c3a6583f3a41"),

			expectedSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("57f1ce10-0b8f-11f0-8383-c3a6583f3a41"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqestHandler: mockReq,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().SummaryCreate(ctx, tt.expectedSummary).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.responseUUID).Return(tt.expectedSummary, nil)

			if tt.activeflowID != uuid.Nil {
				mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.expectedSummary.ActiveflowID, tt.expectedVariables).Return(nil)
			}

			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectedSummary.CustomerID, summary.EventTypeCreated, tt.expectedSummary)

			res, err := h.Create(ctx, tt.customerID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.status, tt.language, tt.content)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedSummary) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedSummary, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseSummary *summary.Summary
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("2dde00e6-0b92-11f0-b3e8-e33b72d0d053"),
			responseSummary: &summary.Summary{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2dde00e6-0b92-11f0-b3e8-e33b72d0d053"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqestHandler: mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().SummaryGet(ctx, tt.id).Return(tt.responseSummary, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseSummary) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseSummary, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[string]string

		responseSummaries []*summary.Summary
	}{
		{
			name: "normal",

			size:  100,
			token: "2025-03-28 21:35:02.809",
			filters: map[string]string{
				"deleted":     "false",
				"customer_id": "2e194dfe-0b92-11f0-b142-1bfcc0d84473",
			},

			responseSummaries: []*summary.Summary{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2e4b9eee-0b92-11f0-9f72-6b6ef6ae4755"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2e7ea3b6-0b92-11f0-85c1-af80bd3dbccf"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqestHandler: mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().SummaryGets(ctx, tt.size, tt.token, tt.filters).Return(tt.responseSummaries, nil)

			res, err := h.Gets(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseSummaries) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseSummaries, res)
			}
		})
	}
}

func Test_GetByCustomerIDAndReferenceIDAndLanguage(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		referenceID uuid.UUID
		language    string

		responseSummaries []*summary.Summary

		expectedFilters map[string]string
		expectedRes     *summary.Summary
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("1b420968-0b93-11f0-a599-eb340fd6a276"),
			referenceID: uuid.FromStringOrNil("1abd614a-0b93-11f0-8711-1b309437dfe1"),
			language:    "en-US",

			responseSummaries: []*summary.Summary{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1aef9584-0b93-11f0-8e2f-b304adfd68a4"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1b1876b6-0b93-11f0-87d8-c3dd1e9297b0"),
					},
				},
			},

			expectedFilters: map[string]string{
				"deleted":      "false",
				"customer_id":  "1b420968-0b93-11f0-a599-eb340fd6a276",
				"reference_id": "1abd614a-0b93-11f0-8711-1b309437dfe1",
				"language":     "en-US",
			},
			expectedRes: &summary.Summary{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1aef9584-0b93-11f0-8e2f-b304adfd68a4"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqestHandler: mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().SummaryGets(ctx, uint64(1000), "", tt.expectedFilters).Return(tt.responseSummaries, nil)

			res, err := h.GetByCustomerIDAndReferenceIDAndLanguage(ctx, tt.customerID, tt.referenceID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseSummary *summary.Summary

		expectedRes *summary.Summary
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("93c8ce56-0ba9-11f0-a274-fb40f9b464d3"),

			responseSummary: &summary.Summary{
				Identity: commonidentity.Identity{},
			},

			expectedRes: &summary.Summary{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("93c8ce56-0ba9-11f0-a274-fb40f9b464d3"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := summaryHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				reqestHandler: mockReq,
			}
			ctx := context.Background()

			mockDB.EXPECT().SummaryDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().SummaryGet(ctx, tt.id).Return(tt.responseSummary, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseSummary.CustomerID, summary.EventTypeDeleted, tt.responseSummary)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseSummary) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseSummary, res)
			}
		})
	}
}

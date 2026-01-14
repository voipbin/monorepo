package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	tmtag "monorepo/bin-tag-manager/models/tag"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_TagCreate(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		tagName string
		detail  string

		response  *tmtag.Tag
		expectRes *tmtag.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			"test1 name",
			"test1 detail",

			&tmtag.Tag{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
				},
			},
			&tmtag.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TagV1TagCreate(ctx, tt.agent.CustomerID, tt.tagName, tt.detail).Return(tt.response, nil)

			res, err := h.TagCreate(ctx, tt.agent, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_TagGets(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		response  []tmtag.Tag
		expectRes []*tmtag.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]tmtag.Tag{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
					},
				},
			},
			[]*tmtag.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
					},
				},
			},
		},
		{
			"2 results",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]tmtag.Tag{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0f620ee-4fbf-11ec-87b2-7372cbac1bb0"),
					},
				},
			},
			[]*tmtag.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0f620ee-4fbf-11ec-87b2-7372cbac1bb0"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TagV1TagGets(ctx, tt.token, tt.size, gomock.Any()).Return(tt.response, nil)

			res, err := h.TagGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TagGet(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		tagID uuid.UUID

		response  *tmtag.Tag
		expectRes *tmtag.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),

			&tmtag.Tag{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&tmtag.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3216dac-4fba-11ec-8551-5b4f1596d5f9"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TagV1TagGet(ctx, tt.tagID).Return(tt.response, nil)

			res, err := h.TagGet(ctx, tt.agent, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_TagDelete(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		tagID uuid.UUID

		resTagGet *tmtag.Tag
		expectRes *tmtag.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),

			&tmtag.Tag{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&tmtag.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TagV1TagGet(ctx, tt.tagID).Return(tt.resTagGet, nil)
			mockReq.EXPECT().TagV1TagDelete(ctx, tt.tagID).Return(tt.resTagGet, nil)

			res, err := h.TagDelete(ctx, tt.agent, tt.tagID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TagUpdate(t *testing.T) {

	tests := []struct {
		name string

		customer *amagent.Agent
		tagID    uuid.UUID
		tagName  string
		detail   string

		resTagGet *tmtag.Tag
		expectRes *tmtag.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
			"test1",
			"detail",

			&tmtag.Tag{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&tmtag.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f829d800-5067-11ec-8370-1b4ec1437594"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().TagV1TagGet(ctx, tt.tagID).Return(tt.resTagGet, nil)
			mockReq.EXPECT().TagV1TagUpdate(ctx, tt.tagID, tt.tagName, tt.detail).Return(tt.resTagGet, nil)

			res, err := h.TagUpdate(ctx, tt.customer, tt.tagID, tt.tagName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

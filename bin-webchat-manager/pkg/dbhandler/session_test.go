package dbhandler

import (
	context "context"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/pkg/cachehandler"
)

func Test_SessionCreate(t *testing.T) {

	tests := []struct {
		name string

		session *session.Session

		responseCurTime *time.Time
		expectRes       *session.Session
	}{
		{
			name: "normal",
			session: &session.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("4baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				WidgetID: uuid.FromStringOrNil("4c8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
				Status:   session.StatusActive,
			},

			responseCurTime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
			expectRes: &session.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("4b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("4baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				WidgetID:       uuid.FromStringOrNil("4c8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
				Status:         session.StatusActive,
				TMLastActivity: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
				TMCreate:       timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
				TMUpdate:       nil,
				TMEnd:          nil,
				TMDelete:       nil,
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().SessionSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().SessionGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.SessionCreate(ctx, tt.session); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.SessionGet(ctx, tt.session.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SessionList(t *testing.T) {
	type test struct {
		name string
		data []*session.Session

		size    uint64
		token   string
		filters map[session.Field]any

		responseCurtime *time.Time
		expectRes       []*session.Session
	}

	widgetID := uuid.FromStringOrNil("5c8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1")
	customerID := uuid.FromStringOrNil("5caabbb2-6b7c-11f0-9f8f-7307c1d1f7ea")

	tests := []test{
		{
			name: "normal",
			data: []*session.Session{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("5d8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
						CustomerID: customerID,
					},
					WidgetID: widgetID,
					Status:   session.StatusActive,
				},
			},

			size:    10,
			token:   "",
			filters: map[session.Field]any{session.FieldWidgetID: widgetID, session.FieldDeleted: false},

			responseCurtime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
			expectRes: []*session.Session{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("5d8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
						CustomerID: customerID,
					},
					WidgetID: widgetID,
					Status:   session.StatusActive,
				},
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

			for _, s := range tt.data {
				mockUtil.EXPECT().TimeNow().Return(tt.responseCurtime)
				mockCache.EXPECT().SessionSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				if err := h.SessionCreate(ctx, s); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.NewUtilHandler().TimeGetCurTime()).AnyTimes()
			res, err := h.SessionList(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if len(res) != len(tt.expectRes) {
				t.Errorf("Wrong match. expect len: %d, got len: %d, res: %v", len(tt.expectRes), len(res), res)
			}
		})
	}
}

func Test_SessionUpdate(t *testing.T) {

	tests := []struct {
		name string

		session *session.Session

		id     uuid.UUID
		fields map[session.Field]any

		responseCurTime *time.Time
		expectRes       *session.Session
	}{
		{
			name: "normal",
			session: &session.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("6baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				WidgetID: uuid.FromStringOrNil("6c8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
				Status:   session.StatusActive,
			},

			id: uuid.FromStringOrNil("6b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
			fields: map[session.Field]any{
				session.FieldStatus: session.StatusEnded,
			},

			responseCurTime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
			expectRes: &session.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("6baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				WidgetID:       uuid.FromStringOrNil("6c8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
				Status:         session.StatusEnded,
				TMLastActivity: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
				TMCreate:       timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
				TMUpdate:       timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().SessionSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().SessionGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.SessionCreate(ctx, tt.session); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.SessionUpdate(ctx, tt.id, tt.fields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.SessionGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_SessionDelete(t *testing.T) {

	tests := []struct {
		name string

		session *session.Session

		id uuid.UUID

		responseCurTime *time.Time
	}{
		{
			name: "normal",
			session: &session.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
					CustomerID: uuid.FromStringOrNil("7baabbb2-6b7c-11f0-9f8f-7307c1d1f7ea"),
				},
				WidgetID: uuid.FromStringOrNil("7c8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),
				Status:   session.StatusActive,
			},

			id: uuid.FromStringOrNil("7b8ab6be-6b7c-11f0-8ec1-8f5b03cd67e1"),

			responseCurTime: timePtr(time.Date(2023, time.February, 15, 3, 22, 17, 994000000, time.UTC)),
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

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().SessionSet(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockCache.EXPECT().SessionGet(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("")).AnyTimes()
			if err := h.SessionCreate(ctx, tt.session); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			if err := h.SessionDelete(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.SessionGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == nil {
				t.Errorf("Wrong match. expect: non-nil TMDelete, got: nil")
			}
		})
	}
}

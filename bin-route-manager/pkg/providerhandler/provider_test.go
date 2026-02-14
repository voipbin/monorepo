package providerhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseProvider *provider.Provider
	}{
		{
			"normal",

			uuid.FromStringOrNil("207afa0a-454a-11ed-9538-03743d74de6a"),

			&provider.Provider{
				ID: uuid.FromStringOrNil("207afa0a-454a-11ed-9538-03743d74de6a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ProviderGet(ctx, tt.id).Return(tt.responseProvider, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseProvider) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProvider, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		providerType provider.Type
		hostname     string
		techPrefix   string
		techPostfix  string
		techHeaders  map[string]string
		providerName string
		detail       string

		responseProvider *provider.Provider
	}{
		{
			"normal",

			provider.TypeSIP,
			"test.com",
			"test_prefix",
			"test_postfix",
			map[string]string{
				"test_header1": "headervalue_1",
				"test_header2": "headervalue_2",
			},
			"test name",
			"test detail",

			&provider.Provider{
				ID: uuid.FromStringOrNil("6c35ba74-454b-11ed-a156-87d4f95324f1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ProviderCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().ProviderGet(ctx, gomock.Any()).Return(tt.responseProvider, nil)
			mockNotify.EXPECT().PublishEvent(ctx, provider.EventTypeProviderCreated, tt.responseProvider)

			res, err := h.Create(ctx, tt.providerType, tt.hostname, tt.techPrefix, tt.techPostfix, tt.techHeaders, tt.providerName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseProvider) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProvider, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		token string
		limit uint64

		responseProviders []*provider.Provider
	}{
		{
			"normal",

			"2020-04-18T03:22:17.995000Z",
			10,

			[]*provider.Provider{
				{
					ID: uuid.FromStringOrNil("efaa7e2a-454c-11ed-9b6d-93874888569d"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			filters := map[provider.Field]any{}
			mockDB.EXPECT().ProviderList(ctx, tt.token, tt.limit, filters).Return(tt.responseProviders, nil)

			res, err := h.List(ctx, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseProviders) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProviders, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseProvider *provider.Provider
	}{
		{
			"normal",

			uuid.FromStringOrNil("35ff4a68-454d-11ed-bf33-cbf62afa54f0"),

			&provider.Provider{
				ID: uuid.FromStringOrNil("35ff4a68-454d-11ed-bf33-cbf62afa54f0"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().ProviderDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ProviderGet(ctx, tt.id).Return(tt.responseProvider, nil)
			mockNotify.EXPECT().PublishEvent(ctx, provider.EventTypeProviderDeleted, tt.responseProvider)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseProvider) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProvider, res)
			}
		})
	}
}

func Test_Update(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		providerType provider.Type
		hostname     string
		techPrefix   string
		techPostfix  string
		techHeaders  map[string]string
		updateName   string
		detail       string

		responseProvider *provider.Provider
	}{
		{
			"normal",

			uuid.FromStringOrNil("eab70c18-4618-11ed-857f-234c1cd0b634"),
			provider.TypeSIP,
			"test.com",
			"test_prefix",
			"test postfix",
			map[string]string{
				"test_header1": "val1",
				"test_header2": "val2",
			},
			"upate name",
			"update detail",

			&provider.Provider{
				ID: uuid.FromStringOrNil("eab70c18-4618-11ed-857f-234c1cd0b634"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &providerHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			fields := map[provider.Field]any{
				provider.FieldType:        tt.providerType,
				provider.FieldHostname:    tt.hostname,
				provider.FieldTechPrefix:  tt.techPrefix,
				provider.FieldTechPostfix: tt.techPostfix,
				provider.FieldTechHeaders: tt.techHeaders,
				provider.FieldName:        tt.updateName,
				provider.FieldDetail:      tt.detail,
			}
			mockDB.EXPECT().ProviderUpdate(ctx, tt.id, fields).Return(nil)
			mockDB.EXPECT().ProviderGet(ctx, tt.id).Return(tt.responseProvider, nil)
			mockNotify.EXPECT().PublishEvent(ctx, provider.EventTypeProviderUpdated, tt.responseProvider)

			res, err := h.Update(ctx, tt.id, tt.providerType, tt.hostname, tt.techPrefix, tt.techPostfix, tt.techHeaders, tt.updateName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseProvider) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseProvider, res)
			}
		})
	}
}

func Test_Get_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("207afa0a-454a-11ed-9538-03743d74de6a")

	mockDB.EXPECT().ProviderGet(ctx, id).Return(nil, fmt.Errorf("database error"))

	res, err := h.Get(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Create_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()

	mockDB.EXPECT().ProviderCreate(ctx, gomock.Any()).Return(fmt.Errorf("database error"))

	res, err := h.Create(ctx, provider.TypeSIP, "test.com", "", "", nil, "name", "detail")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_List_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	filters := map[provider.Field]any{}

	mockDB.EXPECT().ProviderList(ctx, "", uint64(10), filters).Return(nil, fmt.Errorf("database error"))

	res, err := h.List(ctx, "", 10)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Delete_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("35ff4a68-454d-11ed-bf33-cbf62afa54f0")

	mockDB.EXPECT().ProviderDelete(ctx, id).Return(fmt.Errorf("database error"))

	res, err := h.Delete(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Update_Error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("eab70c18-4618-11ed-857f-234c1cd0b634")

	mockDB.EXPECT().ProviderUpdate(ctx, id, gomock.Any()).Return(fmt.Errorf("database error"))

	res, err := h.Update(ctx, id, provider.TypeSIP, "test.com", "", "", nil, "name", "detail")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Create_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()

	mockDB.EXPECT().ProviderCreate(ctx, gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderGet(ctx, gomock.Any()).Return(nil, fmt.Errorf("get error"))

	res, err := h.Create(ctx, provider.TypeSIP, "test.com", "", "", nil, "name", "detail")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Delete_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("35ff4a68-454d-11ed-bf33-cbf62afa54f0")

	mockDB.EXPECT().ProviderDelete(ctx, id).Return(nil)
	mockDB.EXPECT().ProviderGet(ctx, id).Return(nil, fmt.Errorf("get error"))

	res, err := h.Delete(ctx, id)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_Update_GetError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &providerHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}

	ctx := context.Background()
	id := uuid.FromStringOrNil("eab70c18-4618-11ed-857f-234c1cd0b634")

	mockDB.EXPECT().ProviderUpdate(ctx, id, gomock.Any()).Return(nil)
	mockDB.EXPECT().ProviderGet(ctx, id).Return(nil, fmt.Errorf("get error"))

	res, err := h.Update(ctx, id, provider.TypeSIP, "test.com", "", "", nil, "name", "detail")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if res != nil {
		t.Errorf("Expected nil result, got %v", res)
	}
}

func Test_NewProviderHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := NewProviderHandler(mockDB, mockReq, mockNotify)
	if h == nil {
		t.Errorf("Expected handler to be created, got nil")
	}
}

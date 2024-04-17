package providerhandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

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

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		token string
		limit uint64

		responseProviders []*provider.Provider
	}{
		{
			"normal",

			"2020-04-18 03:22:17.995000",
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

			mockDB.EXPECT().ProviderGets(ctx, tt.token, tt.limit).Return(tt.responseProviders, nil)

			res, err := h.Gets(ctx, tt.token, tt.limit)
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

			mockDB.EXPECT().ProviderUpdate(ctx, &provider.Provider{
				ID:          tt.id,
				Type:        tt.providerType,
				Hostname:    tt.hostname,
				TechPrefix:  tt.techPrefix,
				TechPostfix: tt.techPostfix,
				TechHeaders: tt.techHeaders,
				Name:        tt.updateName,
				Detail:      tt.detail,
			}).Return(nil)
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

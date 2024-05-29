package filehandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/dbhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseFile *file.File
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("5f67906c-1531-11ef-acd7-cf9b57d65bcc"),

			responseFile: &file.File{
				ID: uuid.FromStringOrNil("5f67906c-1531-11ef-acd7-cf9b57d65bcc"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &fileHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().FileGet(ctx, tt.id).Return(tt.responseFile, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseFile) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFile, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		token   string
		size    uint64
		filters map[string]string

		responseFiles []*file.File
	}{
		{
			name: "normal",

			token: "2024-05-16 03:22:17.995000",
			size:  10,
			filters: map[string]string{
				"customer_id": "ba5d2ed2-1531-11ef-960b-cfcd7e5676b9",
			},

			responseFiles: []*file.File{
				{
					ID: uuid.FromStringOrNil("e05c0c20-1531-11ef-8c1f-e79b24c34783"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &fileHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().FileGets(ctx, tt.token, tt.size, tt.filters).Return(tt.responseFiles, nil)

			res, err := h.Gets(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseFiles) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFiles, res)
			}
		})
	}
}

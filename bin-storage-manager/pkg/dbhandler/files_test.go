package dbhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/cachehandler"
)

func Test_FileCreate(t *testing.T) {

	tests := []struct {
		name string
		file *file.File

		responseCurTime string
		expectRes       *file.File
	}{
		{
			"normal",
			&file.File{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ee9ff382-13f1-11ef-a41a-b3608f793722"),
					CustomerID: uuid.FromStringOrNil("fb7b9494-13f1-11ef-b22b-13707d54c279"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("fb9db6fa-13f1-11ef-8684-e33adef1ce98"),
				},
				AccountID:        uuid.FromStringOrNil("2e716278-19ad-11ef-b03d-67ce8ed81b5a"),
				ReferenceType:    file.ReferenceTypeRecording,
				ReferenceID:      uuid.FromStringOrNil("305ff91a-1538-11ef-8ceb-f7ad81138af6"),
				Name:             "test name",
				Detail:           "test detail",
				Filename:         "filename.txt",
				Filesize:         1000,
				BucketName:       "bucket_tmp",
				Filepath:         "/tmp/6c0e06ba-146a-11ef-8697-c7c53a81a655",
				URIBucket:        "https://test.com/uri_bucket",
				URIDownload:      "https://test.com/uri_download",
				TMDownloadExpire: "2024-05-18 03:22:17.995000",
			},

			"2024-05-18 03:22:17.995000",
			&file.File{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ee9ff382-13f1-11ef-a41a-b3608f793722"),
					CustomerID: uuid.FromStringOrNil("fb7b9494-13f1-11ef-b22b-13707d54c279"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("fb9db6fa-13f1-11ef-8684-e33adef1ce98"),
				},
				AccountID:        uuid.FromStringOrNil("2e716278-19ad-11ef-b03d-67ce8ed81b5a"),
				ReferenceType:    file.ReferenceTypeRecording,
				ReferenceID:      uuid.FromStringOrNil("305ff91a-1538-11ef-8ceb-f7ad81138af6"),
				Name:             "test name",
				Detail:           "test detail",
				Filename:         "filename.txt",
				BucketName:       "bucket_tmp",
				Filepath:         "/tmp/6c0e06ba-146a-11ef-8697-c7c53a81a655",
				Filesize:         1000,
				URIBucket:        "https://test.com/uri_bucket",
				URIDownload:      "https://test.com/uri_download",
				TMDownloadExpire: "2024-05-18 03:22:17.995000",
				TMCreate:         "2024-05-18 03:22:17.995000",
				TMUpdate:         "9999-01-01T00:00:00.000000Z",
				TMDelete:         "9999-01-01T00:00:00.000000Z",
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
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
			mockCache.EXPECT().FileSet(ctx, gomock.Any())
			if err := h.FileCreate(ctx, tt.file); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FileGet(ctx, tt.file.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().FileSet(ctx, gomock.Any())
			res, err := h.FileGet(ctx, tt.file.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_FileList(t *testing.T) {

	tests := []struct {
		name  string
		files []file.File

		size    uint64
		filters map[file.Field]any

		responseCurTime string
		expectRes       []*file.File
	}{
		{
			"normal",
			[]file.File{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a3f9552a-13f2-11ef-bbbd-23b99b535400"),
						CustomerID: uuid.FromStringOrNil("a42851e0-13f2-11ef-a75e-57b5fc5932e1"),
					},
					Name: "test1",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a45d53cc-13f2-11ef-af26-cbcf0eb06b9e"),
						CustomerID: uuid.FromStringOrNil("a42851e0-13f2-11ef-a75e-57b5fc5932e1"),
					},
					Name: "test2",
				},
			},

			10,
			map[file.Field]any{
				file.FieldCustomerID: uuid.FromStringOrNil("a42851e0-13f2-11ef-a75e-57b5fc5932e1"),
				file.FieldDeleted:    false,
			},

			"2024-05-16 03:22:17.995000",
			[]*file.File{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a3f9552a-13f2-11ef-bbbd-23b99b535400"),
						CustomerID: uuid.FromStringOrNil("a42851e0-13f2-11ef-a75e-57b5fc5932e1"),
					},
					Name:     "test1",
					TMCreate: "2024-05-16 03:22:17.995000",
					TMUpdate: "9999-01-01T00:00:00.000000Z",
					TMDelete: "9999-01-01T00:00:00.000000Z",
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("a45d53cc-13f2-11ef-af26-cbcf0eb06b9e"),
						CustomerID: uuid.FromStringOrNil("a42851e0-13f2-11ef-a75e-57b5fc5932e1"),
					},
					Name:     "test2",
					TMCreate: "2024-05-16 03:22:17.995000",
					TMUpdate: "9999-01-01T00:00:00.000000Z",
					TMDelete: "9999-01-01T00:00:00.000000Z",
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
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}
			ctx := context.Background()

			for _, f := range tt.files {
				mockUtil.EXPECT().TimeGetCurTime().Return(tt.responseCurTime)
				mockCache.EXPECT().FileSet(ctx, gomock.Any())
				if err := h.FileCreate(ctx, &f); err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
			}

			res, err := h.FileList(ctx, utilhandler.TimeGetCurTime(), tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes[0], res[0])
			}
		})
	}
}

func Test_FileUpdate(t *testing.T) {

	tests := []struct {
		name string
		file *file.File

		updateFields map[file.Field]any

		expectRes *file.File
	}{
		{
			"test normal",
			&file.File{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("886d7d76-13f3-11ef-aef8-436d1fc7ffca"),
				},
			},

			map[file.Field]any{
				file.FieldName:   "test name",
				file.FieldDetail: "test detail",
			},

			&file.File{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("886d7d76-13f3-11ef-aef8-436d1fc7ffca"),
				},
				Name:   "test name",
				Detail: "test detail",
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
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().FileSet(ctx, gomock.Any())
			if err := h.FileCreate(ctx, tt.file); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCache.EXPECT().FileSet(ctx, gomock.Any())
			if err := h.FileUpdate(ctx, tt.file.ID, tt.updateFields); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FileGet(ctx, tt.file.ID).Return(nil, fmt.Errorf(""))
			mockCache.EXPECT().FileSet(ctx, gomock.Any())
			res, err := h.FileGet(ctx, tt.file.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res.TMDownloadExpire = ""
			res.TMUpdate = ""
			res.TMCreate = ""
			res.TMDelete = ""
			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_FileDelete(t *testing.T) {

	tests := []struct {
		name string
		file *file.File
	}{
		{
			"normal",
			&file.File{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f435e9b2-13f3-11ef-b332-9374d7dca9d5"),
				},
				Name:   "test file name",
				Detail: "test file detail",
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
				util:  mockUtil,
				db:    dbTest,
				cache: mockCache,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()

			mockCache.EXPECT().FileSet(ctx, gomock.Any())
			if err := h.FileCreate(ctx, tt.file); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FileDel(ctx, tt.file.ID)
			if err := h.FileDelete(ctx, tt.file.ID); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().FileGet(ctx, tt.file.ID).Return(nil, fmt.Errorf("error"))
			mockCache.EXPECT().FileSet(ctx, gomock.Any()).Return(nil)
			res, err := h.FileGet(ctx, tt.file.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.TMDelete == "9999-01-01T00:00:00.000000Z" {
				t.Errorf("Wrong match. expect: any other, got: %s", res.TMDelete)
			}

		})
	}
}

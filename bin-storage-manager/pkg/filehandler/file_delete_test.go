package filehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"
)

var errTest = errors.New("simulated failure")

func Test_Delete_paths(t *testing.T) {
	ctx := context.Background()

	id := uuid.FromStringOrNil("1a2b3c4d-0000-0000-0000-000000000001")
	accountID := uuid.FromStringOrNil("1a2b3c4d-0000-0000-0000-000000000002")
	stored := &file.File{
		Identity:   commonidentity.Identity{ID: id},
		AccountID:  accountID,
		BucketName: "media",
		Filepath:   "bin/obj1",
		Filesize:   321,
	}

	tests := []struct {
		name string

		bucketDeleteStatus int // 0 keeps the fake default

		dbDeleteErr  error
		dbGetErr     error
		decreaseErr  error
		expectBucket bool // whether the bucket object is expected to be deleted
		expectEvent  bool
		expectErr    error
	}{
		{
			name:         "normal",
			expectBucket: true,
			expectEvent:  true,
		},
		{
			name:        "the db delete fails and nothing else happens",
			dbDeleteErr: errTest,
			expectErr:   errTest,
		},
		{
			name:      "the read back fails and the bucket object is kept",
			dbGetErr:  errTest,
			expectErr: errTest,
		},
		{
			name:               "a bucket delete failure is tolerated",
			bucketDeleteStatus: 500,
			expectBucket:       true,
			expectEvent:        true,
		},
		{
			name:         "an account decrease failure is tolerated",
			decreaseErr:  errTest,
			expectBucket: true,
			expectEvent:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			fake := newFakeGCS(t, map[string]int64{"media/bin/obj1": 321})
			if tt.bucketDeleteStatus != 0 {
				fake.deleteStatus["media/bin/obj1"] = tt.bucketDeleteStatus
			}
			h := &fileHandler{db: mockDB, notifyHandler: mockNotify, accountHandler: mockAccount, client: fake.client(t)}

			mockDB.EXPECT().FileDelete(ctx, id).Return(tt.dbDeleteErr)
			if tt.dbDeleteErr == nil {
				if tt.dbGetErr != nil {
					mockDB.EXPECT().FileGet(ctx, id).Return(nil, tt.dbGetErr)
				} else {
					mockDB.EXPECT().FileGet(ctx, id).Return(stored, nil)
				}
			}
			if tt.expectEvent {
				mockNotify.EXPECT().PublishEvent(ctx, file.EventTypeFileDeleted, stored)
				mockAccount.EXPECT().DecreaseFileInfo(ctx, accountID, int64(1), int64(321)).Return(&account.Account{}, tt.decreaseErr)
			}

			res, err := h.Delete(ctx, id)
			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr != nil {
				if res != nil {
					t.Errorf("Wrong match. expect: nil, got: %v", res)
				}
			} else if !reflect.DeepEqual(res, stored) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", stored, res)
			}

			var wantDeleted []string
			if tt.expectBucket {
				wantDeleted = []string{"media/bin/obj1"}
			}
			if got := fake.deletedKeys(); !reflect.DeepEqual(got, wantDeleted) {
				t.Errorf("Wrong match. expect bucket deletes: %v, got: %v", wantDeleted, got)
			}
		})
	}
}

func Test_DeleteBucketfile(t *testing.T) {
	ctx := context.Background()

	fake := newFakeGCS(t, map[string]int64{"media/bin/exists": 5})
	h := &fileHandler{client: fake.client(t)}

	if err := h.DeleteBucketfile(ctx, "media", "bin/exists"); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if got := fake.deletedKeys(); !reflect.DeepEqual(got, []string{"media/bin/exists"}) {
		t.Errorf("Wrong match. got deletes: %v", got)
	}

	// deleting an object that does not exist is reported to the caller.
	if err := h.DeleteBucketfile(ctx, "media", "bin/missing"); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_IsExist(t *testing.T) {
	ctx := context.Background()

	fake := newFakeGCS(t, map[string]int64{"tmp/zip/a.zip": 5})
	h := &fileHandler{client: fake.client(t)}

	tests := []struct {
		name     string
		bucket   string
		filepath string
		expect   bool
	}{
		{"existing object", "tmp", "zip/a.zip", true},
		{"missing object", "tmp", "zip/b.zip", false},
		{"same path in another bucket", "media", "zip/a.zip", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.IsExist(ctx, tt.bucket, tt.filepath); got != tt.expect {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expect, got)
			}
		})
	}
}

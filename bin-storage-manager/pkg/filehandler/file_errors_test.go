package filehandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"
)

var errSimulated = errors.New("simulated failure")

// createFixture wires a fileHandler to mocks and a storage fake for the Create tests.
type createFixture struct {
	h *fileHandler

	mockUtil    *utilhandler.MockUtilHandler
	mockDB      *dbhandler.MockDBHandler
	mockNotify  *notifyhandler.MockNotifyHandler
	mockAccount *accounthandler.MockAccountHandler

	fake *storageFake

	customerID uuid.UUID
	ownerID    uuid.UUID
	fileID     uuid.UUID
	account    *account.Account
}

const (
	testCreateTmpBucket   = "test-tmp"
	testCreateMediaBucket = "test-media"
	testCreateSrcPath     = "tmp/source.wav"
)

func newCreateFixture(t *testing.T, mc *gomock.Controller, fake *storageFake) *createFixture {
	t.Helper()

	f := &createFixture{
		mockUtil:    utilhandler.NewMockUtilHandler(mc),
		mockDB:      dbhandler.NewMockDBHandler(mc),
		mockNotify:  notifyhandler.NewMockNotifyHandler(mc),
		mockAccount: accounthandler.NewMockAccountHandler(mc),
		fake:        fake,

		customerID: uuid.FromStringOrNil("a1b2c3d4-0001-4000-8000-000000000001"),
		ownerID:    uuid.FromStringOrNil("a1b2c3d4-0001-4000-8000-000000000002"),
		fileID:     uuid.FromStringOrNil("a1b2c3d4-0001-4000-8000-000000000003"),
		account:    &account.Account{ID: uuid.FromStringOrNil("a1b2c3d4-0001-4000-8000-000000000004")},
	}

	f.h = &fileHandler{
		utilHandler:    f.mockUtil,
		notifyHandler:  f.mockNotify,
		db:             f.mockDB,
		accountHandler: f.mockAccount,

		client:      newStorageFake(t, fake),
		bucketMedia: testCreateMediaBucket,
		bucketTmp:   testCreateTmpBucket,
	}

	return f
}

func (f *createFixture) dstFilepath() string {
	return fmt.Sprintf("%s/%s", bucketDirectoryBin, f.fileID)
}

func (f *createFixture) create(ctx context.Context) (*file.File, error) {
	return f.h.Create(ctx, f.customerID, f.ownerID, file.ReferenceTypeNormal, f.fileID, file.TypeNone, "name", "detail", "source.wav", testCreateTmpBucket, testCreateSrcPath)
}

// Test_Create_failures checks that each failing step of Create returns the error and stops
// the steps after it. The mocks have no expectations for the later steps, so an event
// publish or an account update after a failure fails the test.
func Test_Create_failures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		sourceExists      bool
		destinationExists bool

		// setup registers the mock expectations of the steps that are reached.
		setup func(f *createFixture)

		expectErr string
	}{
		{
			name:         "source file does not exist",
			sourceExists: false,
			setup:        func(f *createFixture) {},
			expectErr:    "file does not exist",
		},
		{
			name:         "account validation fails",
			sourceExists: true,
			setup: func(f *createFixture) {
				f.mockAccount.EXPECT().ValidateFileInfoByCustomerID(gomock.Any(), f.customerID, int64(1), int64(1)).Return(nil, errSimulated)
			},
			expectErr: "could not pass the account validation",
		},
		{
			name:              "destination already exists",
			sourceExists:      true,
			destinationExists: true,
			setup: func(f *createFixture) {
				f.mockAccount.EXPECT().ValidateFileInfoByCustomerID(gomock.Any(), f.customerID, int64(1), int64(1)).Return(f.account, nil)
				f.mockUtil.EXPECT().UUIDCreate().Return(f.fileID)
			},
			expectErr: "could not move the file",
		},
		{
			name:         "db create fails",
			sourceExists: true,
			setup: func(f *createFixture) {
				f.mockAccount.EXPECT().ValidateFileInfoByCustomerID(gomock.Any(), f.customerID, int64(1), int64(1)).Return(f.account, nil)
				f.mockUtil.EXPECT().UUIDCreate().Return(f.fileID)
				f.mockDB.EXPECT().FileCreate(gomock.Any(), gomock.Any()).Return(errSimulated)
			},
			expectErr: errSimulated.Error(),
		},
		{
			name:         "db get after create fails",
			sourceExists: true,
			setup: func(f *createFixture) {
				f.mockAccount.EXPECT().ValidateFileInfoByCustomerID(gomock.Any(), f.customerID, int64(1), int64(1)).Return(f.account, nil)
				f.mockUtil.EXPECT().UUIDCreate().Return(f.fileID)
				f.mockDB.EXPECT().FileCreate(gomock.Any(), gomock.Any()).Return(nil)
				f.mockDB.EXPECT().FileGet(gomock.Any(), f.fileID).Return(nil, errSimulated)
			},
			expectErr: errSimulated.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			fake := &storageFake{attrs: map[string]bool{}}
			if tt.sourceExists {
				fake.attrs[testCreateTmpBucket+"/"+testCreateSrcPath] = true
			}
			f := newCreateFixture(t, mc, fake)
			if tt.destinationExists {
				fake.attrs[testCreateMediaBucket+"/"+f.dstFilepath()] = true
			}
			tt.setup(f)

			res, err := f.create(ctx)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil file, got: %v", res)
			}
		})
	}
}

// Test_Create_increaseFileInfoFailureIsTolerated checks that a failed account counter update
// is only logged: the file has been stored and the event published by then.
func Test_Create_increaseFileInfoFailureIsTolerated(t *testing.T) {
	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	fake := &storageFake{attrs: map[string]bool{testCreateTmpBucket + "/" + testCreateSrcPath: true}}
	f := newCreateFixture(t, mc, fake)

	stored := &file.File{
		Identity:  commonidentity.Identity{ID: f.fileID},
		AccountID: f.account.ID,
		Filesize:  1,
	}

	f.mockAccount.EXPECT().ValidateFileInfoByCustomerID(gomock.Any(), f.customerID, int64(1), int64(1)).Return(f.account, nil)
	f.mockUtil.EXPECT().UUIDCreate().Return(f.fileID)
	f.mockDB.EXPECT().FileCreate(gomock.Any(), gomock.Any()).Return(nil)
	f.mockDB.EXPECT().FileGet(gomock.Any(), f.fileID).Return(stored, nil)
	f.mockNotify.EXPECT().PublishEvent(gomock.Any(), file.EventTypeFileCreated, stored)
	f.mockAccount.EXPECT().IncreaseFileInfo(gomock.Any(), f.account.ID, int64(1), int64(1)).Return(nil, errSimulated)

	res, err := f.create(ctx)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != stored {
		t.Errorf("Wrong match. expect the stored file, got: %v", res)
	}
	if deleted := fake.deletedKeys(); len(deleted) != 1 || deleted[0] != testCreateTmpBucket+"/"+testCreateSrcPath {
		t.Errorf("Wrong match. expect the source to be moved away, got deleted: %v", deleted)
	}
}

func Test_Get_failures(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("a1b2c3d4-0002-4000-8000-000000000001")

	t.Run("not found is a typed error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &fileHandler{db: mockDB}
		mockDB.EXPECT().FileGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(ctx, id)
		if res != nil {
			t.Errorf("Wrong match. expect: nil file, got: %v", res)
		}

		var typed *cerrors.VoipbinError
		if !errors.As(err, &typed) {
			t.Fatalf("Wrong match. expect: *VoipbinError, got: %v", err)
		}
		if typed.Status != cerrors.StatusNotFound || typed.Reason != "FILE_NOT_FOUND" {
			t.Errorf("Wrong match. expect: not found/FILE_NOT_FOUND, got: %s/%s", typed.Status, typed.Reason)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect the cause to stay reachable, got: %v", err)
		}
	})

	t.Run("other errors are wrapped, not reported as not found", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockDB := dbhandler.NewMockDBHandler(mc)
		h := &fileHandler{db: mockDB}
		mockDB.EXPECT().FileGet(ctx, id).Return(nil, errSimulated)

		res, err := h.Get(ctx, id)
		if res != nil {
			t.Errorf("Wrong match. expect: nil file, got: %v", res)
		}
		if !errors.Is(err, errSimulated) || !strings.Contains(err.Error(), "could not get file info") {
			t.Errorf("Wrong match. expect the wrapped cause, got: %v", err)
		}

		var typed *cerrors.VoipbinError
		if errors.As(err, &typed) {
			t.Errorf("Wrong match. expect: untyped error, got: %v", typed)
		}
	})
}

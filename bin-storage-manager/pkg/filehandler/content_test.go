package filehandler

import (
	"bytes"
	"context"
	stderrors "errors"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-storage-manager/models/file"
)

func Test_CreateFromContent(t *testing.T) {
	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	tmpID := uuid.FromStringOrNil("a1b2c3d4-0001-4000-8000-0000000000aa")
	fake := &storageFake{attrs: map[string]bool{}}
	f := newCreateFixture(t, mc, fake)
	fake.attrs[testCreateTmpBucket+"/tmp/"+tmpID.String()] = true

	stored := &file.File{
		Identity:  commonidentity.Identity{ID: f.fileID},
		AccountID: f.account.ID,
		Filesize:  1,
	}

	gomock.InOrder(
		f.mockUtil.EXPECT().UUIDCreate().Return(tmpID),
		f.mockAccount.EXPECT().ValidateFileInfoByCustomerID(gomock.Any(), f.customerID, int64(1), int64(1)).Return(f.account, nil),
		f.mockUtil.EXPECT().UUIDCreate().Return(f.fileID),
		f.mockDB.EXPECT().FileCreate(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *file.File) error {
			if got.ReferenceType != file.ReferenceTypeNone || got.ReferenceID != uuid.Nil || got.Type != file.TypeMedia || got.Filename != "media-0.jpg" {
				t.Errorf("Wrong match. got: %+v", got)
			}
			return nil
		}),
		f.mockDB.EXPECT().FileGet(gomock.Any(), f.fileID).Return(stored, nil),
		f.mockNotify.EXPECT().PublishEvent(gomock.Any(), file.EventTypeFileCreated, stored),
		f.mockAccount.EXPECT().IncreaseFileInfo(gomock.Any(), f.account.ID, int64(1), int64(1)).Return(f.account, nil),
	)

	res, err := f.h.CreateFromContent(ctx, f.customerID, uuid.Nil, file.TypeMedia, "name", "detail", "media-0.jpg", "image/jpeg", []byte("fake-image-bytes"))
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != stored {
		t.Errorf("Wrong match. expect: %v, got: %v", stored, res)
	}

	if fake.uploaded != 1 || len(fake.uploadBodies) != 1 {
		t.Fatalf("Wrong match. expect: 1 upload, got: %d", fake.uploaded)
	}
	body := fake.uploadBodies[0]
	if !strings.Contains(body, "tmp/"+tmpID.String()) || !strings.Contains(body, "image/jpeg") || !strings.Contains(body, "fake-image-bytes") {
		t.Errorf("Wrong match. the upload does not carry the path, the content type and the bytes")
	}
	if fake.rewrites != 1 {
		t.Errorf("Wrong match. expect: 1 move through Create, got: %d", fake.rewrites)
	}
}

func Test_CreateFromContent_invalid(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		content []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"over the limit", bytes.Repeat([]byte("a"), fileContentMaxSize+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			fake := &storageFake{attrs: map[string]bool{}}
			f := newCreateFixture(t, mc, fake)

			res, err := f.h.CreateFromContent(ctx, f.customerID, uuid.Nil, file.TypeMedia, "n", "d", "f.jpg", "image/jpeg", tt.content)
			var ve *cerrors.VoipbinError
			if res != nil || err == nil || !stderrors.As(err, &ve) || ve.Reason != "FILE_CONTENT_INVALID" {
				t.Errorf("Wrong match. expect: FILE_CONTENT_INVALID, got: %v, %v", res, err)
			}
			if fake.uploaded != 0 {
				t.Errorf("Wrong match. expect: no upload, got: %d", fake.uploaded)
			}
		})
	}
}

func Test_CreateFromContent_maxSizeAccepted(t *testing.T) {
	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	tmpID := uuid.FromStringOrNil("a1b2c3d4-0001-4000-8000-0000000000aa")
	fake := &storageFake{attrs: map[string]bool{}}
	f := newCreateFixture(t, mc, fake)
	fake.attrs[testCreateTmpBucket+"/tmp/"+tmpID.String()] = true

	f.mockUtil.EXPECT().UUIDCreate().Return(tmpID)
	f.mockAccount.EXPECT().ValidateFileInfoByCustomerID(gomock.Any(), f.customerID, int64(1), int64(1)).Return(nil, errSimulated)

	_, err := f.h.CreateFromContent(ctx, f.customerID, uuid.Nil, file.TypeMedia, "n", "d", "f.jpg", "image/jpeg", bytes.Repeat([]byte("a"), fileContentMaxSize))
	if err == nil || !strings.Contains(err.Error(), "could not pass the account validation") {
		t.Fatalf("Wrong match. expect: the account validation error, got: %v", err)
	}
	if fake.uploaded != 1 {
		t.Errorf("Wrong match. expect: the maximum size is uploaded, got: %d", fake.uploaded)
	}
}

func Test_CreateFromContent_failures(t *testing.T) {
	ctx := context.Background()

	t.Run("upload fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		fake := &storageFake{attrs: map[string]bool{}, uploadStatus: 500}
		f := newCreateFixture(t, mc, fake)
		f.mockUtil.EXPECT().UUIDCreate().Return(f.fileID)

		res, err := f.h.CreateFromContent(ctx, f.customerID, uuid.Nil, file.TypeMedia, "n", "d", "f.jpg", "image/jpeg", []byte("x"))
		if res != nil || err == nil || !strings.Contains(err.Error(), "could not") {
			t.Errorf("Wrong match. expect: write error, got: %v, %v", res, err)
		}
		if len(fake.deletedKeys()) != 0 {
			t.Errorf("Wrong match. expect: nothing to delete, got: %v", fake.deletedKeys())
		}
	})

	t.Run("quota fails and the tmp object is deleted", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		tmpID := uuid.FromStringOrNil("a1b2c3d4-0001-4000-8000-0000000000aa")
		fake := &storageFake{attrs: map[string]bool{}}
		f := newCreateFixture(t, mc, fake)
		fake.attrs[testCreateTmpBucket+"/tmp/"+tmpID.String()] = true

		f.mockUtil.EXPECT().UUIDCreate().Return(tmpID)
		f.mockAccount.EXPECT().ValidateFileInfoByCustomerID(gomock.Any(), f.customerID, int64(1), int64(1)).Return(nil, errSimulated)

		res, err := f.h.CreateFromContent(ctx, f.customerID, uuid.Nil, file.TypeMedia, "n", "d", "f.jpg", "image/jpeg", []byte("x"))
		if res != nil || err == nil {
			t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
		}
		deleted := fake.deletedKeys()
		if len(deleted) != 1 || deleted[0] != testCreateTmpBucket+"/tmp/"+tmpID.String() {
			t.Errorf("Wrong match. expect: the tmp object deleted, got: %v", deleted)
		}
	})
}

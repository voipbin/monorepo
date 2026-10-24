package extensionhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func assertExtensionNotFound(t *testing.T, err error) {
	t.Helper()

	var verr *cerrors.VoipbinError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %T %v, want a VoipbinError", err, err)
	}
	if verr.Status != cerrors.StatusNotFound || verr.Reason != "EXTENSION_NOT_FOUND" {
		t.Errorf("error status/reason = %v/%v, want NOT_FOUND/EXTENSION_NOT_FOUND", verr.Status, verr.Reason)
	}
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("the db error must stay reachable through the wrapped cause")
	}
}

func Test_Get_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBBin := dbhandler.NewMockDBHandler(mc)
	h := &extensionHandler{dbBin: mockDBBin}
	ctx := context.Background()
	id := uuid.FromStringOrNil("b38b9d45-f81d-4505-b9ef-9f44da1860cf")

	mockDBBin.EXPECT().ExtensionGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(ctx, id)
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
	assertExtensionNotFound(t, err)
}

func Test_Get_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDBBin := dbhandler.NewMockDBHandler(mc)
	h := &extensionHandler{dbBin: mockDBBin}
	ctx := context.Background()
	id := uuid.FromStringOrNil("b38b9d45-f81d-4505-b9ef-9f44da1860cf")
	dbErr := errors.New("connection reset")

	mockDBBin.EXPECT().ExtensionGet(ctx, id).Return(nil, dbErr)

	res, err := h.Get(ctx, id)
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("error = %v, want the db error", err)
	}
	var verr *cerrors.VoipbinError
	if errors.As(err, &verr) {
		t.Errorf("a generic db error must not be converted to %v", verr.Status)
	}
}

func Test_GetByExtension(t *testing.T) {
	customerID := uuid.FromStringOrNil("a27578e6-756f-45e4-88f0-d97e725f4507")
	want := &extension.Extension{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("b38b9d45-f81d-4505-b9ef-9f44da1860cf"),
			CustomerID: customerID,
		},
		Extension: "1001",
	}
	dbErr := errors.New("connection reset")

	tests := []struct {
		name string

		dbRes *extension.Extension
		dbErr error

		wantRes      *extension.Extension
		wantNotFound bool
		wantDBErr    bool
	}{
		{name: "normal", dbRes: want, wantRes: want},
		{name: "not found", dbErr: dbhandler.ErrNotFound, wantNotFound: true},
		{name: "db error", dbErr: dbErr, wantDBErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDBBin := dbhandler.NewMockDBHandler(mc)
			h := &extensionHandler{dbBin: mockDBBin}
			ctx := context.Background()

			mockDBBin.EXPECT().ExtensionGetByExtension(ctx, customerID, "1001").Return(tt.dbRes, tt.dbErr)

			res, err := h.GetByExtension(ctx, customerID, "1001")
			if !reflect.DeepEqual(res, tt.wantRes) {
				t.Errorf("result = %v, want %v", res, tt.wantRes)
			}

			switch {
			case tt.wantNotFound:
				assertExtensionNotFound(t, err)
			case tt.wantDBErr:
				if !errors.Is(err, dbErr) {
					t.Errorf("error = %v, want the db error", err)
				}
				var verr *cerrors.VoipbinError
				if errors.As(err, &verr) {
					t.Errorf("a generic db error must not be converted to %v", verr.Status)
				}
			default:
				if err != nil {
					t.Errorf("error = %v, want nil", err)
				}
			}
		})
	}
}

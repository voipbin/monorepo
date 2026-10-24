package trunkhandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

func assertTrunkNotFound(t *testing.T, err error) {
	t.Helper()

	var verr *cerrors.VoipbinError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %T %v, want a VoipbinError", err, err)
	}
	if verr.Status != cerrors.StatusNotFound || verr.Reason != "TRUNK_NOT_FOUND" {
		t.Errorf("error status/reason = %v/%v, want NOT_FOUND/TRUNK_NOT_FOUND", verr.Status, verr.Reason)
	}
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("the db error must stay reachable through the wrapped cause")
	}
}

func assertNotConverted(t *testing.T, err, dbErr error) {
	t.Helper()

	if !errors.Is(err, dbErr) {
		t.Errorf("error = %v, want the db error", err)
	}
	var verr *cerrors.VoipbinError
	if errors.As(err, &verr) {
		t.Errorf("a generic db error must not be converted to %v", verr.Status)
	}
}

func Test_Get_cases(t *testing.T) {
	id := uuid.FromStringOrNil("a27578e6-756f-45e4-88f0-d97e725f4507")
	want := &trunk.Trunk{Identity: commonidentity.Identity{ID: id}}
	dbErr := errors.New("connection reset")

	tests := []struct {
		name string

		dbRes *trunk.Trunk
		dbErr error

		wantRes      *trunk.Trunk
		wantNotFound bool
		wantDBErr    bool
	}{
		{name: "not found", dbErr: dbhandler.ErrNotFound, wantNotFound: true},
		{name: "db error", dbErr: dbErr, wantDBErr: true},
		{name: "normal", dbRes: want, wantRes: want},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &trunkHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().TrunkGet(ctx, id).Return(tt.dbRes, tt.dbErr)

			res, err := h.Get(ctx, id)
			if !reflect.DeepEqual(res, tt.wantRes) {
				t.Errorf("result = %v, want %v", res, tt.wantRes)
			}
			switch {
			case tt.wantNotFound:
				assertTrunkNotFound(t, err)
			case tt.wantDBErr:
				assertNotConverted(t, err, dbErr)
			case err != nil:
				t.Errorf("error = %v, want nil", err)
			}
		})
	}
}

func Test_GetByDomainName_cases(t *testing.T) {
	want := &trunk.Trunk{DomainName: "test-trunk"}
	dbErr := errors.New("connection reset")

	tests := []struct {
		name string

		dbRes *trunk.Trunk
		dbErr error

		wantRes      *trunk.Trunk
		wantNotFound bool
		wantDBErr    bool
	}{
		{name: "not found", dbErr: dbhandler.ErrNotFound, wantNotFound: true},
		{name: "db error", dbErr: dbErr, wantDBErr: true},
		{name: "normal", dbRes: want, wantRes: want},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &trunkHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().TrunkGetByDomainName(ctx, "test-trunk").Return(tt.dbRes, tt.dbErr)

			res, err := h.GetByDomainName(ctx, "test-trunk")
			if !reflect.DeepEqual(res, tt.wantRes) {
				t.Errorf("result = %v, want %v", res, tt.wantRes)
			}
			switch {
			case tt.wantNotFound:
				assertTrunkNotFound(t, err)
			case tt.wantDBErr:
				assertNotConverted(t, err, dbErr)
			case err != nil:
				t.Errorf("error = %v, want nil", err)
			}
		})
	}
}

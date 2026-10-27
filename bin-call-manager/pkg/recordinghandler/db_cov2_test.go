package recordinghandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	smfile "monorepo/bin-storage-manager/models/file"

	"monorepo/bin-call-manager/models/recording"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Create_cov2_errors(t *testing.T) {
	recID := uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937")
	afID := uuid.FromStringOrNil("a1b75e8c-0727-11f0-9222-77e37a9b9643")

	tests := []struct {
		name string

		createErr   error
		getErr      error
		variableErr error

		expectGet      bool
		expectVariable bool
		expectErr      string
	}{
		{name: "db create error", createErr: errors.New("c"), expectErr: "could not create the record: c"},
		{name: "db get error", getErr: errors.New("g"), expectGet: true, expectErr: "could not get created"},
		{name: "variable update failure is ignored", variableErr: errors.New("v"), expectGet: true, expectVariable: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &recordingHandler{reqHandler: mockReq, db: mockDB}
			ctx := context.Background()

			stored := &recording.Recording{Identity: commonidentity.Identity{ID: recID}, ActiveflowID: afID}

			mockDB.EXPECT().RecordingCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.expectGet {
				if tt.getErr != nil {
					mockDB.EXPECT().RecordingGet(ctx, recID).Return(nil, tt.getErr)
				} else {
					mockDB.EXPECT().RecordingGet(ctx, recID).Return(stored, nil)
				}
			}
			if tt.expectVariable {
				mockReq.EXPECT().FlowV1VariableSetVariable(ctx, afID, gomock.Any()).Return(tt.variableErr)
			}

			res, err := h.Create(ctx, recID, uuid.Nil, afID, recording.ReferenceTypeCall, uuid.Nil, recording.FormatWAV, uuid.Nil, "name", []string{"f"}, "ast", []string{"c"})
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) || res != nil {
					t.Errorf("Wrong match. expect: %q, got: %v, %v", tt.expectErr, res, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(res, stored) {
				t.Errorf("Wrong match. expect: %v, got: %v, err: %v", stored, res, err)
			}
		})
	}
}

func Test_List_cov2_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &recordingHandler{db: mockDB}
	ctx := context.Background()
	expectErr := errors.New("list")

	mockDB.EXPECT().RecordingList(ctx, uint64(10), "token", gomock.Any()).Return(nil, expectErr)

	res, err := h.List(ctx, 10, "token", map[recording.Field]any{})
	if res != nil || err != expectErr {
		t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", expectErr, res, err)
	}
}

func Test_Get_cov2(t *testing.T) {
	recID := uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937")
	other := errors.New("other")

	tests := []struct {
		name string

		dbErr error

		expectNotFound bool
	}{
		{name: "not found maps to typed error", dbErr: dbhandler.ErrNotFound, expectNotFound: true},
		{name: "other error returned as is", dbErr: other},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &recordingHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().RecordingGet(ctx, recID).Return(nil, tt.dbErr)

			res, err := h.Get(ctx, recID)
			if res != nil || err == nil {
				t.Fatalf("Wrong match. expect error, got: %v/%v", res, err)
			}
			var ve *cerrors.VoipbinError
			if tt.expectNotFound {
				if !errors.As(err, &ve) {
					t.Fatalf("Wrong error type. got: %T", err)
				}
				if ve.Reason != "RECORDING_NOT_FOUND" || !errors.Is(err, dbhandler.ErrNotFound) {
					t.Errorf("Wrong match. reason: %s, err: %v", ve.Reason, err)
				}
				return
			}
			if err != tt.dbErr {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.dbErr, err)
			}
		})
	}
}

func Test_Delete_cov2_errors(t *testing.T) {
	recID := uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937")

	tests := []struct {
		name string

		deleteErr error
		getErr    error
	}{
		{name: "delete error", deleteErr: errors.New("del")},
		{name: "get deleted error", getErr: errors.New("get")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &recordingHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().RecordingDelete(ctx, recID).Return(tt.deleteErr)
			if tt.deleteErr == nil {
				mockDB.EXPECT().RecordingGet(ctx, recID).Return(nil, tt.getErr)
			}

			expect := tt.deleteErr
			if expect == nil {
				expect = tt.getErr
			}
			res, err := h.Delete(ctx, recID)
			if res != nil || err != expect {
				t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", expect, res, err)
			}
		})
	}
}

func Test_deleteRecordingFiles_cov2_errors(t *testing.T) {
	recID := uuid.FromStringOrNil("d5e795ec-612b-11eb-b1f8-87092b928937")
	fileA := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	fileB := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000002")

	t.Run("list error stops", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &recordingHandler{reqHandler: mockReq}

		mockReq.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(1000), gomock.Any()).Return(nil, errors.New("list"))

		h.deleteRecordingFiles(&recording.Recording{Identity: commonidentity.Identity{ID: recID}})
	})

	t.Run("delete error continues to next file", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &recordingHandler{reqHandler: mockReq}

		files := []smfile.File{{Identity: commonidentity.Identity{ID: fileA}}, {Identity: commonidentity.Identity{ID: fileB}}}
		mockReq.EXPECT().StorageV1FileList(gomock.Any(), "", uint64(1000), gomock.Any()).Return(files, nil)
		mockReq.EXPECT().StorageV1FileDelete(gomock.Any(), fileA, 60000).Return(nil, errors.New("del"))
		mockReq.EXPECT().StorageV1FileDelete(gomock.Any(), fileB, 60000).Return(&smfile.File{}, nil)

		h.deleteRecordingFiles(&recording.Recording{Identity: commonidentity.Identity{ID: recID}})
	})
}

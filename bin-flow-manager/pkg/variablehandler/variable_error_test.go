package variablehandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

var errSynthetic = errors.New("synthetic failure")

func Test_Create_error(t *testing.T) {
	id := uuid.FromStringOrNil("58c48f6a-cce2-11ec-9826-af5811eeea09")
	vars := map[string]string{"key1": "val1"}

	tests := []struct {
		name  string
		setup func(mockDB *dbhandler.MockDBHandler)
	}{
		{
			name: "create fails and the stored variable is never read",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().VariableCreate(gomock.Any(), &variable.Variable{ID: id, Variables: vars}).Return(errSynthetic)
			},
		},
		{
			name: "read after create fails",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().VariableCreate(gomock.Any(), &variable.Variable{ID: id, Variables: vars}).Return(nil)
				mockDB.EXPECT().VariableGet(gomock.Any(), id).Return(nil, errSynthetic)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &variableHandler{db: mockDB}
			tt.setup(mockDB)

			res, err := h.Create(context.Background(), id, vars)
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
			if !errors.Is(err, errSynthetic) {
				t.Errorf("Wrong error. expect: %v, got: %v", errSynthetic, err)
			}
		})
	}
}

func Test_Get_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &variableHandler{db: mockDB}
	id := uuid.FromStringOrNil("58c48f6a-cce2-11ec-9826-af5811eeea09")

	mockDB.EXPECT().VariableGet(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(context.Background(), id)
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong error. expect: %v, got: %v", dbhandler.ErrNotFound, err)
	}
}

func Test_Set_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &variableHandler{db: mockDB}
	v := &variable.Variable{ID: uuid.FromStringOrNil("58c48f6a-cce2-11ec-9826-af5811eeea09"), Variables: map[string]string{"k": "v"}}

	mockDB.EXPECT().VariableUpdate(gomock.Any(), v).Return(errSynthetic)

	if err := h.Set(context.Background(), v); !errors.Is(err, errSynthetic) {
		t.Errorf("Wrong error. expect: %v, got: %v", errSynthetic, err)
	}
}

func Test_SetVariable_error(t *testing.T) {
	id := uuid.FromStringOrNil("58c48f6a-cce2-11ec-9826-af5811eeea09")

	tests := []struct {
		name  string
		setup func(mockDB *dbhandler.MockDBHandler)
	}{
		{
			name: "read fails and nothing is written",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().VariableGet(gomock.Any(), id).Return(nil, errSynthetic)
			},
		},
		{
			name: "write fails after the new value is substituted",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().VariableGet(gomock.Any(), id).Return(&variable.Variable{ID: id, Variables: map[string]string{"base": "b1"}}, nil)
				mockDB.EXPECT().VariableUpdate(gomock.Any(), &variable.Variable{ID: id, Variables: map[string]string{"base": "b1", "k1": "b1-x"}}).Return(errSynthetic)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &variableHandler{db: mockDB}
			tt.setup(mockDB)

			err := h.SetVariable(context.Background(), id, map[string]string{"k1": "${base}-x"})
			if !errors.Is(err, errSynthetic) {
				t.Errorf("Wrong error. expect: %v, got: %v", errSynthetic, err)
			}
		})
	}
}

func Test_DeleteVariable_error(t *testing.T) {
	id := uuid.FromStringOrNil("58c48f6a-cce2-11ec-9826-af5811eeea09")

	tests := []struct {
		name  string
		setup func(mockDB *dbhandler.MockDBHandler)
	}{
		{
			name: "read fails and nothing is written",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().VariableGet(gomock.Any(), id).Return(nil, errSynthetic)
			},
		},
		{
			name: "write fails after the key is removed",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().VariableGet(gomock.Any(), id).Return(&variable.Variable{ID: id, Variables: map[string]string{"keep": "1", "drop": "2"}}, nil)
				mockDB.EXPECT().VariableUpdate(gomock.Any(), &variable.Variable{ID: id, Variables: map[string]string{"keep": "1"}}).Return(errSynthetic)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &variableHandler{db: mockDB}
			tt.setup(mockDB)

			if err := h.DeleteVariable(context.Background(), id, "drop"); !errors.Is(err, errSynthetic) {
				t.Errorf("Wrong error. expect: %v, got: %v", errSynthetic, err)
			}
		})
	}
}

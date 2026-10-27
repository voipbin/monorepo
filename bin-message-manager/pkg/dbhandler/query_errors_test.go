package dbhandler

import (
	"context"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commondatabasehandler "monorepo/bin-common-handler/pkg/databasehandler"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
)

// failingValuer is a driver.Valuer whose value can never be built.
type failingValuer struct{}

func (failingValuer) Value() (driver.Value, error) {
	return nil, errors.New("simulated value failure")
}

// failingSqlizer is a string based value that squirrel treats as a sqlizer and that cannot build its sql.
type failingSqlizer string

func (failingSqlizer) ToSql() (string, []interface{}, error) {
	return "", nil, errors.New("simulated sql failure")
}

var _ sq.Sqlizer = failingSqlizer("")

// outOfRangeTime is a time that json cannot encode (the year must be in [0,9999]).
func outOfRangeTime() *time.Time {
	tm := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	return &tm
}

// Test_queryBuildErrors covers the failures before any statement is executed. The database handle is closed,
// so reaching the execution step would produce a different error than the expected prefix.
func Test_queryBuildErrors(t *testing.T) {
	id := uuid.FromStringOrNil("b2c3d4e5-b0b3-11ee-a1b2-0f3d5a7b9c21")
	curTime := time.Date(2021, 2, 26, 18, 26, 49, 0, time.UTC)

	tests := []struct {
		name string

		fn func(h *handler) error

		expectNow    int
		expectPrefix string
	}{
		{
			name: "create with targets that cannot be encoded",
			fn: func(h *handler) error {
				m := &message.Message{Targets: []target.Target{{TMUpdate: outOfRangeTime()}}}
				m.ID = id
				return h.MessageCreate(context.Background(), m)
			},
			expectNow:    1,
			expectPrefix: "could not prepare fields. MessageCreate",
		},
		{
			name: "update with targets that cannot be encoded",
			fn: func(h *handler) error {
				return h.MessageUpdate(context.Background(), id, map[message.Field]any{
					message.FieldTargets: []target.Target{{TMUpdate: outOfRangeTime()}},
				})
			},
			expectNow:    1,
			expectPrefix: "could not prepare fields. MessageUpdate",
		},
		{
			name: "update with a value whose sql cannot be built",
			fn: func(h *handler) error {
				return h.MessageUpdate(context.Background(), id, map[message.Field]any{
					message.FieldText: failingSqlizer("x"),
				})
			},
			expectNow:    1,
			expectPrefix: "could not build query. MessageUpdate",
		},
		{
			name: "list with a filter whose value cannot be built",
			fn: func(h *handler) error {
				_, err := h.MessageList(context.Background(), pathCurTime, 10, map[message.Field]any{
					message.FieldText: commondatabasehandler.NotEq{Value: failingValuer{}},
				})
				return err
			},
			expectPrefix: "could not build query. MessageList",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockUtil, _ := newErrorPathHandler(mc, newClosedDB(t))

			if tt.expectNow > 0 {
				mockUtil.EXPECT().TimeNow().Return(&curTime).Times(tt.expectNow)
			}

			err := tt.fn(h)
			if err == nil || !strings.HasPrefix(err.Error(), tt.expectPrefix) {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.expectPrefix, err)
			}
		})
	}
}

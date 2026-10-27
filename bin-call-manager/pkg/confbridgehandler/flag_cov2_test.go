package confbridgehandler

import (
	"context"
	stderrors "errors"
	"reflect"
	"testing"

	"monorepo/bin-call-manager/models/confbridge"
)

func Test_FlagAdd_cov2(t *testing.T) {
	ctx := context.Background()

	withFlag := cov2CB(func(c *confbridge.Confbridge) { c.Flags = []confbridge.Flag{confbridge.FlagNoAutoLeave} })

	tests := []struct {
		name string

		setup func(m *cov2Mocks)

		expectRes *confbridge.Confbridge
		expectErr bool
	}{
		{
			name: "get failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectErr: true,
		},
		{
			name: "flag already exists",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(withFlag, nil)
			},
			expectRes: withFlag,
		},
		{
			name: "update flags failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(cov2CB(nil), nil)
				m.db.EXPECT().ConfbridgeSetFlags(ctx, cov2CBID, []confbridge.Flag{confbridge.FlagNoAutoLeave}).Return(errCov2)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			res, err := h.FlagAdd(ctx, cov2CBID, confbridge.FlagNoAutoLeave)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr && !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_FlagRemove_cov2(t *testing.T) {
	ctx := context.Background()

	noFlag := cov2CB(nil)
	withFlag := cov2CB(func(c *confbridge.Confbridge) { c.Flags = []confbridge.Flag{confbridge.FlagNoAutoLeave} })

	tests := []struct {
		name string

		setup func(m *cov2Mocks)

		expectRes *confbridge.Confbridge
		expectErr bool
	}{
		{
			name: "get failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(nil, errCov2)
			},
			expectErr: true,
		},
		{
			name: "flag does not exist",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(noFlag, nil)
			},
			expectRes: noFlag,
		},
		{
			name: "update flags failed",
			setup: func(m *cov2Mocks) {
				m.db.EXPECT().ConfbridgeGet(ctx, cov2CBID).Return(withFlag, nil)
				m.db.EXPECT().ConfbridgeSetFlags(ctx, cov2CBID, []confbridge.Flag{}).Return(errCov2)
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, m := newCov2Handler(t)
			tt.setup(m)

			res, err := h.FlagRemove(ctx, cov2CBID, confbridge.FlagNoAutoLeave)
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr && !stderrors.Is(err, errCov2) {
				t.Errorf("Wrong match. expect: wrapped errCov2, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

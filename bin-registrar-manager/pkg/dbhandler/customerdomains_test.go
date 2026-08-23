package dbhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	reflect "reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

func Test_isDuplicateEntryErr(t *testing.T) {
	tests := []struct {
		name string

		err error

		expectRes bool
	}{
		{
			name:      "nil error",
			err:       nil,
			expectRes: false,
		},
		{
			name:      "mysql duplicate entry",
			err:       &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'ab12' for key 'ux_registrar_customer_domains_domain_label'"},
			expectRes: true,
		},
		{
			name:      "mysql other error",
			err:       &mysql.MySQLError{Number: 1054, Message: "Unknown column"},
			expectRes: false,
		},
		{
			name:      "wrapped mysql duplicate entry",
			err:       fmt.Errorf("exec failed: %w", &mysql.MySQLError{Number: 1062, Message: "Duplicate entry"}),
			expectRes: true,
		},
		{
			name:      "sqlite unique constraint",
			err:       stderrors.New("UNIQUE constraint failed: registrar_customer_domains.domain_label"),
			expectRes: true,
		},
		{
			name:      "duplicate entry string",
			err:       stderrors.New("Error 1062: Duplicate entry 'x' for key 'y'"),
			expectRes: true,
		},
		{
			name:      "unrelated error",
			err:       stderrors.New("connection refused"),
			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := isDuplicateEntryErr(tt.err); res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerDomainCreate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC); return &t }()

	tests := []struct {
		name string

		customerDomain *customerdomain.CustomerDomain

		responseCurTime *time.Time
		expectRes       *customerdomain.CustomerDomain
	}{
		{
			"have all",

			&customerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("1a4b3c2d-7f80-11ee-8f5a-1b2c3d4e5f60"),
				DomainLabel: "ab12",
				Realm:       "ab12.reg.voipbin.net",
			},

			curTime,
			&customerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("1a4b3c2d-7f80-11ee-8f5a-1b2c3d4e5f60"),
				DomainLabel: "ab12",
				Realm:       "ab12.reg.voipbin.net",
				TMCreate:    curTime,
				TMUpdate:    nil,
			},
		},
		{
			"backfilled uuid label",

			&customerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("2b5c4d3e-7f80-11ee-8f5a-1b2c3d4e5f60"),
				DomainLabel: "2b5c4d3e-7f80-11ee-8f5a-1b2c3d4e5f60",
				Realm:       "2b5c4d3e-7f80-11ee-8f5a-1b2c3d4e5f60.registrar.voipbin.net",
			},

			curTime,
			&customerdomain.CustomerDomain{
				CustomerID:  uuid.FromStringOrNil("2b5c4d3e-7f80-11ee-8f5a-1b2c3d4e5f60"),
				DomainLabel: "2b5c4d3e-7f80-11ee-8f5a-1b2c3d4e5f60",
				Realm:       "2b5c4d3e-7f80-11ee-8f5a-1b2c3d4e5f60.registrar.voipbin.net",
				TMCreate:    curTime,
				TMUpdate:    nil,
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
				utilHandler: mockUtil,
				db:          dbTest,
				cache:       mockCache,
			}

			ctx := context.Background()

			mockUtil.EXPECT().TimeNow().Return(tt.responseCurTime)
			mockCache.EXPECT().CustomerDomainSet(ctx, gomock.Any())
			if err := h.CustomerDomainCreate(ctx, tt.customerDomain); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := h.CustomerDomainGet(ctx, tt.customerDomain.CustomerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_CustomerDomainCreate_duplicate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()
	curTime := func() *time.Time { t := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC); return &t }()

	first := &customerdomain.CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("3c6d5e4f-7f80-11ee-8f5a-1b2c3d4e5f60"),
		DomainLabel: "cd34",
		Realm:       "cd34.reg.voipbin.net",
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().CustomerDomainSet(ctx, gomock.Any())
	if err := h.CustomerDomainCreate(ctx, first); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// same domain_label for a different customer must return ErrDuplicate
	dupLabel := &customerdomain.CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("4d7e6f50-7f80-11ee-8f5a-1b2c3d4e5f60"),
		DomainLabel: "cd34",
		Realm:       "other.reg.voipbin.net",
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	err := h.CustomerDomainCreate(ctx, dupLabel)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if !stderrors.Is(err, ErrDuplicate) {
		t.Errorf("Wrong match. expect: ErrDuplicate, got: %v", err)
	}

	// same customer_id must return ErrDuplicate as well (concurrent-create race)
	dupCustomer := &customerdomain.CustomerDomain{
		CustomerID:  first.CustomerID,
		DomainLabel: "ef56",
		Realm:       "ef56.reg.voipbin.net",
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	err = h.CustomerDomainCreate(ctx, dupCustomer)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if !stderrors.Is(err, ErrDuplicate) {
		t.Errorf("Wrong match. expect: ErrDuplicate, got: %v", err)
	}
}

func Test_CustomerDomainGet_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()

	_, err := h.CustomerDomainGet(ctx, uuid.FromStringOrNil("99999999-7f80-11ee-8f5a-1b2c3d4e5f60"))
	if !stderrors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func Test_CustomerDomainGetByRealm(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC); return &t }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()

	cd := &customerdomain.CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("5e8f7061-7f80-11ee-8f5a-1b2c3d4e5f60"),
		DomainLabel: "gh78",
		Realm:       "gh78.reg.voipbin.net",
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().CustomerDomainSet(ctx, gomock.Any())
	if err := h.CustomerDomainCreate(ctx, cd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := &customerdomain.CustomerDomain{
		CustomerID:  cd.CustomerID,
		DomainLabel: "gh78",
		Realm:       "gh78.reg.voipbin.net",
		TMCreate:    curTime,
		TMUpdate:    nil,
	}

	// cache miss -> DB hit -> cache set
	mockCache.EXPECT().CustomerDomainGetByRealm(ctx, cd.Realm).Return(nil, fmt.Errorf("cache miss"))
	mockCache.EXPECT().CustomerDomainSet(ctx, gomock.Any())
	res, err := h.CustomerDomainGetByRealm(ctx, cd.Realm)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectRes, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}

	// cache hit -> no DB access needed
	mockCache.EXPECT().CustomerDomainGetByRealm(ctx, cd.Realm).Return(expectRes, nil)
	res, err = h.CustomerDomainGetByRealm(ctx, cd.Realm)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !reflect.DeepEqual(expectRes, res) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}

	// unknown realm -> ErrNotFound
	mockCache.EXPECT().CustomerDomainGetByRealm(ctx, "nope.reg.voipbin.net").Return(nil, fmt.Errorf("cache miss"))
	_, err = h.CustomerDomainGetByRealm(ctx, "nope.reg.voipbin.net")
	if !stderrors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func Test_CustomerDomainUpdate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC); return &t }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()

	cd := &customerdomain.CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("6f908172-7f80-11ee-8f5a-1b2c3d4e5f60"),
		DomainLabel: "6f908172-7f80-11ee-8f5a-1b2c3d4e5f60",
		Realm:       "6f908172-7f80-11ee-8f5a-1b2c3d4e5f60.registrar.voipbin.net",
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().CustomerDomainSet(ctx, gomock.Any())
	if err := h.CustomerDomainCreate(ctx, cd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// update: the OLD realm cache key must be invalidated, the new row re-cached
	fields := map[customerdomain.Field]any{
		customerdomain.FieldDomainLabel: "ij90",
		customerdomain.FieldRealm:       "ij90.reg.voipbin.net",
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().CustomerDomainDelByRealm(ctx, "6f908172-7f80-11ee-8f5a-1b2c3d4e5f60.registrar.voipbin.net").Return(nil)
	mockCache.EXPECT().CustomerDomainSet(ctx, gomock.Any())
	if err := h.CustomerDomainUpdate(ctx, cd.CustomerID, fields); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.CustomerDomainGet(ctx, cd.CustomerID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.DomainLabel != "ij90" || res.Realm != "ij90.reg.voipbin.net" {
		t.Errorf("Wrong match. expect: ij90/ij90.reg.voipbin.net, got: %s/%s", res.DomainLabel, res.Realm)
	}
	if res.TMUpdate == nil {
		t.Errorf("Wrong match. expect: tm_update set, got: nil")
	}

	// empty fields is a no-op
	if err := h.CustomerDomainUpdate(ctx, cd.CustomerID, map[customerdomain.Field]any{}); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	// update of a missing row returns an error
	mockUpdateErr := h.CustomerDomainUpdate(ctx, uuid.FromStringOrNil("77777777-7f80-11ee-8f5a-1b2c3d4e5f60"), fields)
	if !stderrors.Is(mockUpdateErr, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", mockUpdateErr)
	}
}

func Test_CustomerDomainUpdate_duplicate(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC); return &t }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()

	first := &customerdomain.CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("8aa19283-7f80-11ee-8f5a-1b2c3d4e5f60"),
		DomainLabel: "kl12",
		Realm:       "kl12.reg.voipbin.net",
	}
	second := &customerdomain.CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("9bb2a394-7f80-11ee-8f5a-1b2c3d4e5f60"),
		DomainLabel: "mn34",
		Realm:       "mn34.reg.voipbin.net",
	}

	for _, cd := range []*customerdomain.CustomerDomain{first, second} {
		mockUtil.EXPECT().TimeNow().Return(curTime)
		mockCache.EXPECT().CustomerDomainSet(ctx, gomock.Any())
		if err := h.CustomerDomainCreate(ctx, cd); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	// updating second to first's label collides -> ErrDuplicate
	fields := map[customerdomain.Field]any{
		customerdomain.FieldDomainLabel: "kl12",
		customerdomain.FieldRealm:       "kl12-other.reg.voipbin.net",
	}
	mockUtil.EXPECT().TimeNow().Return(curTime)
	err := h.CustomerDomainUpdate(ctx, second.CustomerID, fields)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: ok")
	}
	if !stderrors.Is(err, ErrDuplicate) {
		t.Errorf("Wrong match. expect: ErrDuplicate, got: %v", err)
	}
}

func Test_CustomerDomainDelete(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC); return &t }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()

	cd := &customerdomain.CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("acc3b4a5-7f80-11ee-8f5a-1b2c3d4e5f60"),
		DomainLabel: "op56",
		Realm:       "op56.reg.voipbin.net",
	}

	mockUtil.EXPECT().TimeNow().Return(curTime)
	mockCache.EXPECT().CustomerDomainSet(ctx, gomock.Any())
	if err := h.CustomerDomainCreate(ctx, cd); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// hard delete + realm cache invalidation
	mockCache.EXPECT().CustomerDomainDelByRealm(ctx, cd.Realm).Return(nil)
	if err := h.CustomerDomainDelete(ctx, cd.CustomerID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	_, err := h.CustomerDomainGet(ctx, cd.CustomerID)
	if !stderrors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}

	// deleting a missing row returns ErrNotFound (idempotency handled by the handler layer)
	err = h.CustomerDomainDelete(ctx, cd.CustomerID)
	if !stderrors.Is(err, ErrNotFound) {
		t.Errorf("Wrong match. expect: ErrNotFound, got: %v", err)
	}
}

func Test_CustomerDomainList(t *testing.T) {

	curTime := func() *time.Time { t := time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC); return &t }()

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := handler{
		utilHandler: mockUtil,
		db:          dbTest,
		cache:       mockCache,
	}

	ctx := context.Background()

	domains := []*customerdomain.CustomerDomain{
		{
			CustomerID:  uuid.FromStringOrNil("bdd4c5b6-7f80-11ee-8f5a-1b2c3d4e5f60"),
			DomainLabel: "qr78",
			Realm:       "qr78.reg.voipbin.net",
		},
		{
			CustomerID:  uuid.FromStringOrNil("cee5d6c7-7f80-11ee-8f5a-1b2c3d4e5f60"),
			DomainLabel: "st90",
			Realm:       "st90.reg.voipbin.net",
		},
	}
	for _, cd := range domains {
		mockUtil.EXPECT().TimeNow().Return(curTime)
		mockCache.EXPECT().CustomerDomainSet(ctx, gomock.Any())
		if err := h.CustomerDomainCreate(ctx, cd); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
	}

	res, err := h.CustomerDomainList(ctx, 100, "2026-08-24 00:00:00.000000", map[customerdomain.Field]any{})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) < 2 {
		t.Errorf("Wrong match. expect: >= 2 rows, got: %d", len(res))
	}

	// filter by realm
	res, err = h.CustomerDomainList(ctx, 100, "2026-08-24 00:00:00.000000", map[customerdomain.Field]any{
		customerdomain.FieldRealm: "qr78.reg.voipbin.net",
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if len(res) != 1 || res[0].DomainLabel != "qr78" {
		t.Errorf("Wrong match. expect: single qr78 row, got: %v", res)
	}
}

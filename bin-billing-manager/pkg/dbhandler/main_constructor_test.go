package dbhandler

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-billing-manager/pkg/cachehandler"
)

func Test_NewHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)

	res := NewHandler(db, mockCache)
	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler, got: %T", res)
	}
	if h.db != db || h.cache != mockCache || h.utilHandler == nil {
		t.Errorf("Wrong match. every dependency must be wired. got: %+v", h)
	}
}

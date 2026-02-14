package dbhandler

import (
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/pkg/cachehandler"

	"github.com/DATA-DOG/go-sqlmock"
	"go.uber.org/mock/gomock"
)

func Test_NewDBHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("could not create mock: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	mc := gomock.NewController(t)
	defer mc.Finish()
	mockCache := cachehandler.NewMockCacheHandler(mc)

	h := NewDBHandler(db, mockCache)
	if h == nil {
		t.Fatal("expected handler, got nil")
	}

	dbh, ok := h.(*dbHandler)
	if !ok {
		t.Fatal("handler is not dbHandler type")
	}

	if dbh.db == nil {
		t.Error("db should not be nil")
	}
	if dbh.cache == nil {
		t.Error("cache should not be nil")
	}
	if dbh.util == nil {
		t.Error("util should not be nil")
	}

	// verify util is correctly initialized
	if _, ok := dbh.util.(utilhandler.UtilHandler); !ok {
		t.Error("util should be UtilHandler type")
	}
}

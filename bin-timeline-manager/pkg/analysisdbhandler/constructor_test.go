package analysisdbhandler

import (
	"database/sql"
	"testing"
)

func Test_NewAnalysisDBHandler(t *testing.T) {
	db := &sql.DB{}

	res := NewAnalysisDBHandler(db)

	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler, got: %T", res)
	}
	if h.db != db {
		t.Errorf("Wrong match. the database is not the injected one")
	}
	if h.utilHandler == nil {
		t.Errorf("Wrong match. expect: a util handler, got: nil")
	}
}

package actionhandler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

func TestActionPatchGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `[{"type":"hangup"}]`)
	}))
	defer ts.Close()
	targetURL := ts.URL

	h := &actionHandler{}

	tests := []struct {
		name   string
		act    *action.Action
		callID uuid.UUID
	}{
		{
			"normal",
			&action.Action{
				ID:     uuid.FromStringOrNil("6e2a0cee-fba2-11ea-a469-a350f2dad844"),
				Option: []byte(fmt.Sprintf(`{"event_url": "%s"}`, targetURL)),
			},
			uuid.FromStringOrNil("549d358a-fbfc-11ea-a625-43073fda56b9"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := h.ActionPatchGet(tt.act, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

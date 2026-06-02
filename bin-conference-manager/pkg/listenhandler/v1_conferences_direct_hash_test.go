package listenhandler

import (
	"net/http"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	pkgerrors "github.com/pkg/errors"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conference-manager/pkg/conferencehandler"
	"monorepo/bin-conference-manager/pkg/dbhandler"
)

func Test_processV1ConferencesIDDirectHashRegenerate_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		sockHandler:       mockSock,
		conferenceHandler: mockConf,
	}

	req := &sock.Request{
		URI:    "/v1/conferences/a1b2c3d4-0000-0000-0000-000000000001/direct-hash-regenerate",
		Method: sock.RequestMethodPost,
	}

	mockConf.EXPECT().
		DirectHashRegenerate(gomock.Any(), gomock.Any()).
		Return(nil, pkgerrors.Wrap(dbhandler.ErrNotFound, "conference not found"))

	res, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
	}
}

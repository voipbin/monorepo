package listenhandler

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-schedule-manager/pkg/backuphandler"
)

func setupBackupListenHandler(t *testing.T) (*gomock.Controller, *listenHandler, *backuphandler.MockBackupHandler) {
	mc := gomock.NewController(t)

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockBackup := backuphandler.NewMockBackupHandler(mc)

	h := &listenHandler{
		sockHandler: mockSock,

		backupHandler: mockBackup,
	}

	return mc, h, mockBackup
}

func Test_processV1BackupsPost(t *testing.T) {
	mc, h, mockBackup := setupBackupListenHandler(t)
	defer mc.Finish()

	responseResult := &backuphandler.Result{
		Path:  "/var/backups/voipbin/voipbin-20260801T020000Z.sql.gz",
		Bytes: 10485760,
	}

	mockBackup.EXPECT().Backup(gomock.Any()).Return(responseResult, nil)

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/backups",
		Method: sock.RequestMethodPost,
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	expectRes := &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       []byte(`{"path":"/var/backups/voipbin/voipbin-20260801T020000Z.sql.gz","bytes":10485760}`),
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expectRes, res)
	}
}

func Test_processV1BackupsPost_error(t *testing.T) {
	mc, h, mockBackup := setupBackupListenHandler(t)
	defer mc.Finish()

	mockBackup.EXPECT().Backup(gomock.Any()).Return(nil, fmt.Errorf("backup dir is not configured"))

	res, err := h.processRequest(&sock.Request{
		URI:    "/v1/backups",
		Method: sock.RequestMethodPost,
	})
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong match. expect: 500, got: %d", res.StatusCode)
	}
}

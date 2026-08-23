package dbhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/utilhandler"

	_ "github.com/mattn/go-sqlite3"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-registrar-manager/pkg/cachehandler"
)

func insertTestContact(t *testing.T, id string, endpoint string) {
	t.Helper()

	if _, err := dbTest.Exec(
		"insert into ps_contacts(id, uri, expiration_time, qualify_frequency, outbound_proxy, path, user_agent, qualify_timeout, reg_server, authenticate_qualify, via_addr, via_port, call_id, endpoint, prune_on_boot) values(?, 'sip:test@1.2.3.4:5060', 1613498199, 0, '', '', 'test-ua', 3.0, 'reg-server', 'no', '1.2.3.4', 5060, 'call-id', ?, 'no')",
		id, endpoint,
	); err != nil {
		t.Fatalf("could not insert the test contact. err: %v", err)
	}
}

func countContactsByEndpoint(t *testing.T, endpoint string) int {
	t.Helper()

	var count int
	if err := dbTest.QueryRow("select count(*) from ps_contacts where endpoint = ?", endpoint).Scan(&count); err != nil {
		t.Fatalf("could not count the contacts. err: %v", err)
	}
	return count
}

func Test_AstContactDeleteByEndpoint(t *testing.T) {
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

	targetEndpoint := "1001@old-realm-uuid.registrar.voipbin.net"
	otherEndpoint := "1002@keep-realm.registrar.voipbin.net"

	insertTestContact(t, "contact-1", targetEndpoint)
	insertTestContact(t, "contact-2", targetEndpoint)
	insertTestContact(t, "contact-3", otherEndpoint)

	// deletes ALL rows of the endpoint and invalidates the contact cache
	mockCache.EXPECT().AstContactsDel(ctx, targetEndpoint).Return(nil)
	if err := h.AstContactDeleteByEndpoint(ctx, targetEndpoint); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if count := countContactsByEndpoint(t, targetEndpoint); count != 0 {
		t.Errorf("Wrong match. expect: 0, got: %d", count)
	}

	// other endpoints are untouched
	if count := countContactsByEndpoint(t, otherEndpoint); count != 1 {
		t.Errorf("Wrong match. expect: 1, got: %d", count)
	}

	// deleting a non-existent endpoint is a no-op (idempotent re-run)
	mockCache.EXPECT().AstContactsDel(ctx, "nobody@nowhere.registrar.voipbin.net").Return(nil)
	if err := h.AstContactDeleteByEndpoint(ctx, "nobody@nowhere.registrar.voipbin.net"); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

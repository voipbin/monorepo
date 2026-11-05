package main

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/carriertrackhandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
)

// The adapter package does not import mcptoolhandler, so this is the one place
// the compiler checks that the handler satisfies the interface it is wired to.
var _ mcptoolhandler.CarrierHandler = (*carriertrackhandler.Handler)(nil)

const wiringKeys = "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

func wiringServer(mc *gomock.Controller, url string) (*dbhandler.MockDBHandler, uuid.UUID) {
	id := uuid.Must(uuid.NewV4())
	server := &mcpserver.McpServer{
		Identity: commonidentity.Identity{ID: id, CustomerID: uuid.Must(uuid.NewV4())},
		URL:      url,
		Status:   mcpserver.StatusActive,
		AuthType: mcpserver.AuthTypeBearer,
	}
	db := dbhandler.NewMockDBHandler(mc)
	db.EXPECT().McpServerGet(gomock.Any(), id).Return(server, nil).AnyTimes()
	return db, id
}

// Test_carrierWiring_onlyTheOpenCarrierIsReachable runs the production wiring
// (the default table and the real adapter handler) against records on the
// reserved host. The servers hold no saved key, so a call ends before any
// request could be made. Only the open carrier lists a tool; every other
// reserved url is closed and ends with the fixed text.
func Test_carrierWiring_onlyTheOpenCarrierIsReachable(t *testing.T) {
	table := mcpserver.DefaultCarrierTable()
	if len(table) != 1 || table["/fedex"].Carrier != "fedex" {
		t.Fatalf("the production table must hold the fedex line only, got %+v", table)
	}

	carriers, err := carriertrackhandler.New(10)
	if err != nil {
		t.Fatal(err)
	}

	closed := []string{
		"https://carrier.invalid/ups", "https://carrier.invalid/dhl",
		"https://CARRIER.invalid./dhl", "https://carrier\u3002invalid/ups", "https://carrier.invalid/dhl?x=1", "https://carrier.invalid:443/dhl",
		"https://carrier.invalid/fedex?x=1", "https://carrier.invalid:443/fedex", "https://carrier.invalid/fedex/",
	}
	for _, url := range closed {
		t.Run("closed "+url, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			db, id := wiringServer(mc, url)
			tool, err := mcptoolhandler.NewMcpToolHandler(db, wiringKeys, 5, nil, table, carriers)
			if err != nil {
				t.Fatal(err)
			}

			if tools, err := tool.ListTools(context.Background(), id); err == nil || len(tools) != 0 {
				t.Errorf("no tool may be listed, got %+v %v", tools, err)
			}
			for _, name := range []string{"track_dhl", "track_fedex"} {
				text, isError, err := tool.CallTool(context.Background(), id, name, `{"tracking_number":"ABCD12345678"}`)
				if err != nil || !isError || text != mcpserver.IntegrationUnavailableText {
					t.Errorf("%s: got %q %v %v", name, text, isError, err)
				}
			}
		})
	}

	// the spelling of the host does not matter (case, trailing dots, an
	// ideographic full stop), the same record is the same carrier
	for _, url := range []string{"https://CARRIER.invalid./fedex", "https://carrier\u3002invalid/fedex"} {
		t.Run("open "+url, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			db, id := wiringServer(mc, url)
			tool, err := mcptoolhandler.NewMcpToolHandler(db, wiringKeys, 5, nil, table, carriers)
			if err != nil {
				t.Fatal(err)
			}
			if tools, err := tool.ListTools(context.Background(), id); err != nil || len(tools) != 1 || tools[0].Name != "track_fedex" {
				t.Errorf("expected the fedex tool, got %+v %v", tools, err)
			}
		})
	}

	t.Run("open https://carrier.invalid/fedex", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		db, id := wiringServer(mc, "https://carrier.invalid/fedex")
		tool, err := mcptoolhandler.NewMcpToolHandler(db, wiringKeys, 5, nil, table, carriers)
		if err != nil {
			t.Fatal(err)
		}

		tools, err := tool.ListTools(context.Background(), id)
		if err != nil || len(tools) != 1 || tools[0].Name != "track_fedex" {
			t.Fatalf("expected the fedex tool, got %+v %v", tools, err)
		}

		text, isError, err := tool.CallTool(context.Background(), id, "track_fedex", `{"tracking_number":"ABCD12345678"}`)
		if want := carriertrackhandler.Message(carriertrackhandler.CodeCredentialNotConfigured, "fedex", mcpserver.CredentialClientIDSecret); err != nil || !isError || text != want {
			t.Errorf("a server without a saved key must end with %q, got %q %v %v", want, text, isError, err)
		}

		text, isError, err = tool.CallTool(context.Background(), id, "track_dhl", `{"tracking_number":"ABCD12345678"}`)
		if want := carriertrackhandler.Message(carriertrackhandler.CodeToolUnknown, "fedex", mcpserver.CredentialClientIDSecret); err != nil || !isError || text != want {
			t.Errorf("another tool name must end with %q, got %q %v %v", want, text, isError, err)
		}
	})
}

// countingCarriers records the calls it receives and answers none of them.
type countingCarriers struct {
	tools atomic.Int32
	calls atomic.Int32
}

func (c *countingCarriers) Tools(string) ([]mcpserver.CarrierToolDef, bool) {
	c.tools.Add(1)
	return nil, false
}

func (c *countingCarriers) Call(context.Context, mcpserver.CarrierCall) (string, bool, error) {
	c.calls.Add(1)
	return "", false, nil
}

// Test_carrierWiring_aCarrierWithoutItsLineIsClosed removes the fedex line
// from the table (the adapter stays registered) and checks that the record is
// closed and that the adapter side is never reached, which is what keeps the
// saved key from being sent anywhere.
func Test_carrierWiring_aCarrierWithoutItsLineIsClosed(t *testing.T) {
	table := mcpserver.DefaultCarrierTable()
	delete(table, "/fedex")

	mc := gomock.NewController(t)
	defer mc.Finish()

	db, id := wiringServer(mc, "https://carrier.invalid/fedex")
	counter := &countingCarriers{}
	tool, err := mcptoolhandler.NewMcpToolHandler(db, wiringKeys, 5, nil, table, counter)
	if err != nil {
		t.Fatal(err)
	}

	if tools, err := tool.ListTools(context.Background(), id); err == nil || len(tools) != 0 {
		t.Errorf("no tool may be listed, got %+v %v", tools, err)
	}
	text, isError, err := tool.CallTool(context.Background(), id, "track_fedex", `{"tracking_number":"ABCD12345678"}`)
	if err != nil || !isError || text != mcpserver.IntegrationUnavailableText {
		t.Errorf("got %q %v %v", text, isError, err)
	}
	if counter.tools.Load() != 0 || counter.calls.Load() != 0 {
		t.Errorf("the adapter side was reached: %d, %d", counter.tools.Load(), counter.calls.Load())
	}
}

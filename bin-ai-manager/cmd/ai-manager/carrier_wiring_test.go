package main

import (
	"context"
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

// Test_carrierWiring_productionTableIsEmptyAndEveryReservedUrlIsClosed runs the
// production wiring (the default table and the real adapter handler, which has
// no adapter yet) against records on the reserved host: none may list a tool or
// reach a carrier, and a call must end with the fixed text.
func Test_carrierWiring_productionTableIsEmptyAndEveryReservedUrlIsClosed(t *testing.T) {
	table := mcpserver.DefaultCarrierTable()
	if len(table) != 0 {
		t.Fatalf("the production table must be empty, has %d entries", len(table))
	}

	carriers, err := carriertrackhandler.New(10)
	if err != nil {
		t.Fatal(err)
	}

	for _, url := range []string{
		"https://carrier.invalid/ups", "https://carrier.invalid/fedex", "https://carrier.invalid/dhl",
		"https://CARRIER.invalid./dhl", "https://carrier\u3002invalid/ups", "https://carrier.invalid/dhl?x=1", "https://carrier.invalid:443/dhl",
	} {
		t.Run(url, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			id := uuid.Must(uuid.NewV4())
			server := &mcpserver.McpServer{
				Identity: commonidentity.Identity{ID: id, CustomerID: uuid.Must(uuid.NewV4())},
				URL:      url,
				Status:   mcpserver.StatusActive,
				AuthType: mcpserver.AuthTypeBearer,
			}
			db := dbhandler.NewMockDBHandler(mc)
			db.EXPECT().McpServerGet(gomock.Any(), id).Return(server, nil).AnyTimes()

			tool, err := mcptoolhandler.NewMcpToolHandler(db, "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=", 5, nil, table, carriers)
			if err != nil {
				t.Fatal(err)
			}

			if tools, err := tool.ListTools(context.Background(), id); err == nil || len(tools) != 0 {
				t.Errorf("no tool may be listed, got %+v %v", tools, err)
			}
			text, isError, err := tool.CallTool(context.Background(), id, "track_dhl", `{"tracking_number":"ABCD12345678"}`)
			if err != nil || !isError || text != mcpserver.IntegrationUnavailableText {
				t.Errorf("got %q %v %v", text, isError, err)
			}
		})
	}
}

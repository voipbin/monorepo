package main

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/carriertrackhandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/mcpschema"
	"monorepo/bin-ai-manager/pkg/mcptoolhandler"
)

// The adapter package does not import mcptoolhandler, so this is the one place
// the compiler checks that the handler satisfies the interface it is wired to.
var _ mcptoolhandler.CarrierHandler = (*carriertrackhandler.Handler)(nil)

var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

const wiringKeys = "1:MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

// wiringTool builds the tool handler the process builds, over one server
// record that has no saved key. A call with no key ends before any network:
// the tracking number is checked first, then the key, and there is none.
func wiringTool(t *testing.T, table mcpserver.CarrierTable, carriers mcptoolhandler.CarrierHandler, url string) (mcptoolhandler.McpToolHandler, uuid.UUID) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	id := uuid.Must(uuid.NewV4())
	server := &mcpserver.McpServer{
		Identity: commonidentity.Identity{ID: id, CustomerID: uuid.Must(uuid.NewV4())},
		URL:      url,
		Status:   mcpserver.StatusActive,
		AuthType: mcpserver.AuthTypeBearer,
	}
	db := dbhandler.NewMockDBHandler(mc)
	db.EXPECT().McpServerGet(gomock.Any(), id).Return(server, nil).AnyTimes()

	tool, err := mcptoolhandler.NewMcpToolHandler(db, wiringKeys, 5, nil, table, carriers)
	if err != nil {
		t.Fatal(err)
	}
	return tool, id
}

// Test_carrierWiring_openUrlsListTheToolOfTheCarrier runs the production wiring
// (the default table and the real adapter handler) against the urls that name
// a carrier of the table. The tool is listed, its definition is acceptable to
// the model providers, and a call without a saved key ends with the fixed text.
// No call here carries a key, so none can reach a carrier.
func Test_carrierWiring_openUrlsListTheToolOfTheCarrier(t *testing.T) {
	table := mcpserver.DefaultCarrierTable()
	if len(table) != 3 || table["/dhl"].Carrier != "dhl" || table["/fedex"].Carrier != "fedex" || table["/ups"].Carrier != "ups" {
		t.Fatalf("the production table must hold the dhl, fedex and ups lines, got %+v", table)
	}
	carriers, err := carriertrackhandler.New(10)
	if err != nil {
		t.Fatal(err)
	}

	for path, entry := range table {
		for _, host := range []string{"carrier.invalid", "CARRIER.invalid.", "carrier\u3002invalid"} {
			url := "https://" + host + path
			t.Run(url, func(t *testing.T) {
				tool, id := wiringTool(t, table, carriers, url)

				tools, err := tool.ListTools(context.Background(), id)
				if err != nil || len(tools) != 1 {
					t.Fatalf("the open carrier must list one tool, got %+v %v", tools, err)
				}
				def := tools[0]
				if def.Name != "track_"+entry.Carrier {
					t.Errorf("tool name = %q", def.Name)
				}
				if entry.Carrier == "ups" && !strings.Contains(def.Description, "1Z") {
					t.Errorf("the ups tool must describe the 1Z tracking number, got %q", def.Description)
				}

				// A name the sessions accept: [A-Za-z0-9_-], at most 51 bytes.
				if len(def.Name) == 0 || len(def.Name) > 51 || !toolNamePattern.MatchString(def.Name) {
					t.Errorf("tool name %q is not accepted by the sessions", def.Name)
				}

				// The schema passes the provider-neutral normalizer unchanged.
				var schema map[string]any
				if err := json.Unmarshal(def.InputSchema, &schema); err != nil {
					t.Fatal(err)
				}
				norm, rep := mcpschema.Normalize(schema, 64<<10)
				if rep.ToolDropped || rep.DroppedKeys != 0 || rep.DroppedPropsN != 0 || rep.Rewrites != 0 {
					t.Errorf("the schema was changed by the normalizer: %+v", rep)
				}
				before, _ := json.Marshal(schema)
				after, _ := json.Marshal(norm)
				if string(before) != string(after) {
					t.Errorf("the schema changed: %s -> %s", before, after)
				}

				// No saved key: the call ends before any network.
				number := "ABCD12345678"
				if entry.Carrier == "ups" {
					number = "1Z999AA10123456784"
				}
				text, isError, err := tool.CallTool(context.Background(), id, def.Name, `{"tracking_number":"`+number+`"}`)
				if err != nil || !isError || text != carriertrackhandler.Message(carriertrackhandler.CodeCredentialNotConfigured, entry.Carrier, entry.Credential) {
					t.Errorf("got %q %v %v", text, isError, err)
				}
			})
		}
	}
}

// Test_carrierWiring_closedUrlsReachNothing runs the production wiring against
// urls that do not name a line of the table: none lists a tool or reaches a
// carrier, and a call ends with the fixed text. The urls use paths no carrier
// owns, and variants of an open path that are not its canonical form.
func Test_carrierWiring_closedUrlsReachNothing(t *testing.T) {
	table := mcpserver.DefaultCarrierTable()
	carriers, err := carriertrackhandler.New(10)
	if err != nil {
		t.Fatal(err)
	}

	urls := []string{
		"https://carrier.invalid/fake", "https://carrier\u3002invalid/fake",
		"https://carrier.invalid/unlisted", "https://CARRIER.invalid./unlisted", "https://carrier\u3002invalid/unlisted",
		"https://carrier.invalid/DHL", "https://carrier.invalid/%64hl", "https://carrier.invalid/FEDEX", "https://carrier.invalid/UPS",
	}
	for open := range table {
		urls = append(urls,
			"https://carrier.invalid"+open+"?x=1", "https://carrier.invalid:443"+open, "https://carrier.invalid"+open+"/")
	}

	for _, url := range urls {
		t.Run(url, func(t *testing.T) {
			tool, id := wiringTool(t, table, carriers, url)

			if tools, err := tool.ListTools(context.Background(), id); err == nil || len(tools) != 0 {
				t.Errorf("no tool may be listed, got %+v %v", tools, err)
			}
			names := []string{"track_fake"}
			for _, entry := range table {
				names = append(names, "track_"+entry.Carrier)
			}
			for _, name := range names {
				text, isError, err := tool.CallTool(context.Background(), id, name, `{"tracking_number":"ABCD12345678"}`)
				if err != nil || !isError || text != mcpserver.IntegrationUnavailableText {
					t.Errorf("%s: got %q %v %v", name, text, isError, err)
				}
			}
		})
	}
}

// Test_carrierWiring_anotherCarrierToolNameIsRefused checks that the tool of
// one open carrier is not reachable through the record of another one.
func Test_carrierWiring_anotherCarrierToolNameIsRefused(t *testing.T) {
	table := mcpserver.DefaultCarrierTable()
	carriers, err := carriertrackhandler.New(10)
	if err != nil {
		t.Fatal(err)
	}

	for path, entry := range table {
		for _, other := range table {
			if other.Carrier == entry.Carrier {
				continue
			}
			t.Run(entry.Carrier+" record, "+other.Carrier+" tool", func(t *testing.T) {
				tool, id := wiringTool(t, table, carriers, "https://carrier.invalid"+path)
				text, isError, err := tool.CallTool(context.Background(), id, "track_"+other.Carrier, `{"tracking_number":"ABCD12345678"}`)
				if want := carriertrackhandler.Message(carriertrackhandler.CodeToolUnknown, entry.Carrier, entry.Credential); err != nil || !isError || text != want {
					t.Errorf("another tool name must end with %q, got %q %v %v", want, text, isError, err)
				}
			})
		}
	}
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

// Test_carrierWiring_aCarrierWithoutItsLineIsClosed removes the line of each
// open carrier from the table (the adapter stays registered) and checks that
// the record is closed and that the adapter side is never reached, which is
// what keeps the saved key from being sent anywhere.
func Test_carrierWiring_aCarrierWithoutItsLineIsClosed(t *testing.T) {
	for path, entry := range mcpserver.DefaultCarrierTable() {
		t.Run(entry.Carrier, func(t *testing.T) {
			table := mcpserver.DefaultCarrierTable()
			delete(table, path)

			counter := &countingCarriers{}
			tool, id := wiringTool(t, table, counter, "https://carrier.invalid"+path)

			if tools, err := tool.ListTools(context.Background(), id); err == nil || len(tools) != 0 {
				t.Errorf("no tool may be listed, got %+v %v", tools, err)
			}
			text, isError, err := tool.CallTool(context.Background(), id, "track_"+entry.Carrier, `{"tracking_number":"ABCD12345678"}`)
			if err != nil || !isError || text != mcpserver.IntegrationUnavailableText {
				t.Errorf("got %q %v %v", text, isError, err)
			}
			if counter.tools.Load() != 0 || counter.calls.Load() != 0 {
				t.Errorf("the adapter side was reached: %d, %d", counter.tools.Load(), counter.calls.Load())
			}
		})
	}
}

// Test_carrierWiring_aCarrierIsClosedByRemovingItsLine shows that taking the
// line out of the table closes the carrier even when the adapter is still
// registered and the server holds a saved key: nothing is listed and nothing
// is called. (The other half, a line without an adapter, is closed by the
// adapter handler itself and is tested in carriertrackhandler and
// mcptoolhandler.)
func Test_carrierWiring_aCarrierIsClosedByRemovingItsLine(t *testing.T) {
	carriers, err := carriertrackhandler.New(10)
	if err != nil {
		t.Fatal(err)
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	// The record holds a saved key; the closed path never decrypts it.
	id := uuid.Must(uuid.NewV4())
	server := &mcpserver.McpServer{
		Identity:         commonidentity.Identity{ID: id, CustomerID: uuid.Must(uuid.NewV4())},
		URL:              "https://carrier.invalid/dhl",
		Status:           mcpserver.StatusActive,
		AuthType:         mcpserver.AuthTypeBearer,
		SecretCiphertext: []byte("not a real ciphertext"),
		SecretNonce:      []byte("123456789012"),
		KeyVersion:       1,
	}
	db := dbhandler.NewMockDBHandler(mc)
	db.EXPECT().McpServerGet(gomock.Any(), id).Return(server, nil).AnyTimes()

	// The table without the line of the carrier.
	tool, err := mcptoolhandler.NewMcpToolHandler(db, wiringKeys, 5, nil, mcpserver.CarrierTable{}, carriers)
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
}

// Test_carrierWiring_everyOtherReservedUrlIsClosed runs the production wiring
// against reserved-host urls the table does not list, and against an empty and
// a nil table: none may list a tool or reach a carrier, and a call must end
// with the fixed text.
func Test_carrierWiring_everyOtherReservedUrlIsClosed(t *testing.T) {
	carriers, err := carriertrackhandler.New(10)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		table mcpserver.CarrierTable
		url   string
	}{
		{"unlisted path", mcpserver.DefaultCarrierTable(), "https://carrier.invalid/unlisted"},
		{"unlisted path, odd host", mcpserver.DefaultCarrierTable(), "https://CARRIER.invalid./unlisted"},
		{"unlisted path, ideographic stop", mcpserver.DefaultCarrierTable(), "https://carrier\u3002invalid/unlisted"},
		{"empty table", mcpserver.CarrierTable{}, "https://carrier.invalid/ups"},
		{"empty table, odd host", mcpserver.CarrierTable{}, "https://CARRIER.invalid./ups"},
		{"nil table", nil, "https://carrier.invalid/ups"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, id := wiringTool(t, tt.table, carriers, tt.url)

			if tools, err := tool.ListTools(context.Background(), id); err == nil || len(tools) != 0 {
				t.Errorf("no tool may be listed, got %+v %v", tools, err)
			}
			for _, name := range []string{"track_dhl", "track_fedex", "track_ups"} {
				text, isError, err := tool.CallTool(context.Background(), id, name, `{"tracking_number":"ABCD12345678"}`)
				if err != nil || !isError || text != mcpserver.IntegrationUnavailableText {
					t.Errorf("%s: got %q %v %v", name, text, isError, err)
				}
			}
		})
	}
}

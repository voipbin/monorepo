package requesthandler

import (
	"context"
	"testing"

	amteam "monorepo/bin-ai-manager/models/team"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
)

func Test_AIV1TeamWrappers(t *testing.T) {
	teamID := uuid.FromStringOrNil("1e3a5c7b-9d2f-4a60-8b14-3c5e7a9b1d28")
	customerID := uuid.FromStringOrNil("5a7c9e1b-3d4f-4c82-a036-7e9b1d3f5a40")
	memberID := uuid.FromStringOrNil("9c1e3a5b-7d6f-4e04-b258-1a3c5e7b9d62")
	aiID := uuid.FromStringOrNil("d3f5a7c9-1b8e-4a26-8c7a-5e7b9d1f3a84")

	teamWant := &amteam.Team{Identity: identity.Identity{ID: teamID}, Name: "support"}
	const teamResponse = `{"id":"1e3a5c7b-9d2f-4a60-8b14-3c5e7a9b1d28","name":"support"}`

	newName := "renamed"
	members := []amteam.Member{{ID: memberID, Name: "first", AIID: aiID}}
	params := map[string]any{"mode": "x"}

	cases := []rpcCase{
		{
			name:           "list",
			queue:          aiQueue,
			expectURI:      "/v1/teams?page_token=2020-09-20T03%3A23%3A20.995000Z&page_size=10",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"5a7c9e1b-3d4f-4c82-a036-7e9b1d3f5a40"}`,
			responseData:   `[` + teamResponse + `]`,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1TeamList(context.Background(), "2020-09-20T03:23:20.995000Z", 10, map[amteam.Field]any{amteam.FieldCustomerID: customerID})
			},
			expectRes: []amteam.Team{*teamWant},
		},
		{
			name:           "get",
			queue:          aiQueue,
			expectURI:      "/v1/teams/1e3a5c7b-9d2f-4a60-8b14-3c5e7a9b1d28",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   teamResponse,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1TeamGet(context.Background(), teamID)
			},
			expectRes: teamWant,
		},
		{
			name:           "create",
			queue:          aiQueue,
			expectURI:      "/v1/teams",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"5a7c9e1b-3d4f-4c82-a036-7e9b1d3f5a40","name":"support","detail":"d","start_member_id":"9c1e3a5b-7d6f-4e04-b258-1a3c5e7b9d62","members":[{"id":"9c1e3a5b-7d6f-4e04-b258-1a3c5e7b9d62","name":"first","ai_id":"d3f5a7c9-1b8e-4a26-8c7a-5e7b9d1f3a84"}],"parameter":{"mode":"x"}}`,
			responseData:   teamResponse,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1TeamCreate(context.Background(), customerID, "support", "d", memberID, members, params)
			},
			expectRes: teamWant,
		},
		{
			name:           "direct_hash_regenerate",
			queue:          aiQueue,
			expectURI:      "/v1/teams/1e3a5c7b-9d2f-4a60-8b14-3c5e7a9b1d28/direct-hash-regenerate",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeNone,
			responseData:   `{"id":"1e3a5c7b-9d2f-4a60-8b14-3c5e7a9b1d28","direct_hash":"abc123"}`,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1TeamDirectHashRegenerate(context.Background(), teamID)
			},
			expectRes: &amteam.Team{Identity: identity.Identity{ID: teamID}, DirectHash: "abc123"},
		},
		{
			name:           "delete",
			queue:          aiQueue,
			expectURI:      "/v1/teams/1e3a5c7b-9d2f-4a60-8b14-3c5e7a9b1d28",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   teamResponse,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1TeamDelete(context.Background(), teamID)
			},
			expectRes: teamWant,
		},
		{
			name:           "update_name_and_members_only",
			queue:          aiQueue,
			expectURI:      "/v1/teams/1e3a5c7b-9d2f-4a60-8b14-3c5e7a9b1d28",
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     `{"name":"renamed","members":[{"id":"9c1e3a5b-7d6f-4e04-b258-1a3c5e7b9d62","name":"first","ai_id":"d3f5a7c9-1b8e-4a26-8c7a-5e7b9d1f3a84"}]}`,
			responseData:   teamResponse,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1TeamUpdate(context.Background(), teamID, &newName, nil, nil, &members, nil)
			},
			expectRes: teamWant,
		},
		{
			name:           "update_start_member_and_parameter",
			queue:          aiQueue,
			expectURI:      "/v1/teams/1e3a5c7b-9d2f-4a60-8b14-3c5e7a9b1d28",
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     `{"start_member_id":"9c1e3a5b-7d6f-4e04-b258-1a3c5e7b9d62","parameter":{"mode":"x"}}`,
			responseData:   teamResponse,
			call: func(h *requestHandler) (any, error) {
				return h.AIV1TeamUpdate(context.Background(), teamID, nil, nil, &memberID, nil, &params)
			},
			expectRes: teamWant,
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "list_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.AIV1TeamList(ctx, "", 10, map[amteam.Field]any{amteam.FieldCustomerID: make(chan int)})
	})
	runMarshalErrorCase(t, "create_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.AIV1TeamCreate(ctx, customerID, "support", "d", memberID, nil, map[string]any{"mode": make(chan int)})
	})
	runMarshalErrorCase(t, "update_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		bad := map[string]any{"mode": make(chan int)}
		return h.AIV1TeamUpdate(ctx, teamID, nil, nil, nil, nil, &bad)
	})
}

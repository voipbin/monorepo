# Add Direct SIP Handlers for AI, AI Team, and Agent Resource Types

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix SIP 404 errors when calling `sip:direct.<hash>@sip.voipbin.net` for AI, AI team, and agent resources by adding missing handler cases in call-manager's direct SIP routing.

**Architecture:** The switch statement in `startIncomingDomainTypeSIPDirect()` only handles `"extension"` and `"conference"` resource types. We add a new `TypeAI` address type in `bin-common-handler`, then add three new cases (`"ai"`, `"ai_team"`, `"agent"`) each with a dedicated handler function that looks up the resource, creates a temporary flow with appropriate actions, and starts the call. AI/AI team use `Answer` + `AITalk` actions (like conference pattern). Agent uses `Connect` action (like extension pattern).

**Tech Stack:** Go, gomock, table-driven tests, RabbitMQ RPC via requesthandler

---

## Context

**Root cause:** `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go:96-105` — the switch in `startIncomingDomainTypeSIPDirect()` only handles `"extension"` and `"conference"`. When `resource_type` is `"ai"`, `"ai_team"`, or `"agent"`, it falls through to `default` → hangup with `ChannelCauseNoRouteDestination` (SIP 404).

**Key files:**
- Address model: `bin-common-handler/models/address/main.go`
- Implementation: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go`
- Tests: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_test.go`

**Existing patterns to follow:**
- `startIncomingDomainTypeSIPDirectExtension()` (lines 108-152): Fetches extension → creates flow with `[TypeConnect]` → starts call, destination `TypeExtension`
- `startIncomingDomainTypeSIPDirectConference()` (lines 154-196): Fetches conference → creates flow with `[TypeAnswer, TypeConferenceJoin]` → starts call, destination `TypeConference`

**New imports needed in implementation file:**
- `amaicall "monorepo/bin-ai-manager/models/aicall"` — for `AssistanceTypeAI`, `AssistanceTypeTeam`

**New imports needed in test file:**
- `amai "monorepo/bin-ai-manager/models/ai"` — for `AIV1AIGet` return type
- `amaicall "monorepo/bin-ai-manager/models/aicall"` — for `AssistanceTypeAI`, `AssistanceTypeTeam`
- `amteam "monorepo/bin-ai-manager/models/team"` — for `AIV1TeamGet` return type
- `amagent "monorepo/bin-agent-manager/models/agent"` — for `AgentV1AgentGet` return type

**RequestHandler methods available:**
- `AIV1AIGet(ctx, aiID uuid.UUID) (*amai.AI, error)`
- `AIV1TeamGet(ctx, teamID uuid.UUID) (*amteam.Team, error)`
- `AgentV1AgentGet(ctx, agentID uuid.UUID) (*amagent.Agent, error)`
- `FlowV1FlowCreate(ctx, customerID, flowType, name, detail, actions, onCompleteFlowID, persist)`

---

### Task 0: Add `TypeAI` address type to bin-common-handler

**Files:**
- Modify: `bin-common-handler/models/address/main.go`

**Step 1: Add the constant**

Add `TypeAI` to the constants block in `bin-common-handler/models/address/main.go`, alphabetically after `TypeAgent`:

```go
const (
	TypeNone       Type = ""           // no type specified
	TypeAgent      Type = "agent"      // target is agent's id.
	TypeAI         Type = "ai"         // target is AI resource's id
	TypeConference Type = "conference" // target is conference's id
	TypeEmail      Type = "email"      // target is email address
	TypeExtension  Type = "extension"  // target is extension
	TypeLine       Type = "line"       // target is naver line's id
	TypeSIP        Type = "sip"        // target is sip destination
	TypeTel        Type = "tel"        // target tel number
)
```

**Step 2: Run bin-common-handler verification**

Run: `cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
Expected: PASS (additive constant only, no existing code affected)

---

### Task 1: Add `"ai"` handler — write failing test

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_test.go`

**Step 1: Write the failing test**

Add `Test_startIncomingDomainTypeSIP_directAI` to the test file, after the existing `Test_startIncomingDomainTypeSIP_directExtension` test (after line 476). Follow the exact same pattern as the extension test but for AI:

```go
func Test_startIncomingDomainTypeSIP_directAI(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource *commonaddress.Address
		responseDirect *dmdirect.Direct
		responseAI     *amai.AI
		responseFlow   *fmflow.Flow
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.400",

				DestinationNumber: "direct.e90e5ce89f4d",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("27b03598-34df-46af-8b37-8d1ee0d7439d"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "ai",
				ResourceID:   uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
				Hash:         "e90e5ce89f4d",
			},
			responseAI: &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-ai",
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1f2f3f4-f5f6-f7f8-f9fa-fbfcfdfeff01"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.e90e5ce89f4d").Return(tt.responseDirect, nil)
			mockReq.EXPECT().AIV1AIGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseAI, nil)

			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type: fmaction.TypeAITalk,
					Option: fmaction.ConvertOption(fmaction.OptionAITalk{
						AssistanceType: amaicall.AssistanceTypeAI,
						AssistanceID:   tt.responseAI.ID,
					}),
				},
			}

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseAI.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				fmt.Sprintf("tmp flow for direct ai call. ai_id: %s", tt.responseAI.ID),
				expectActions,
				uuid.Nil,
				false,
			).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseAI.CustomerID).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseAI.CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

New imports needed in the test file (add to existing import block):
```go
amai "monorepo/bin-ai-manager/models/ai"
amaicall "monorepo/bin-ai-manager/models/aicall"
```

**Step 2: Run test to verify it fails**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP_directAI`
Expected: FAIL — the `"ai"` case doesn't exist yet, so `startIncomingDomainTypeSIPDirect` hits the `default` case and calls `HangingUp` instead of `AIV1AIGet`.

---

### Task 2: Implement `"ai"` handler

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go`

**Step 1: Add imports**

Add to the import block:
```go
amaicall "monorepo/bin-ai-manager/models/aicall"
```

**Step 2: Add the switch case**

In `startIncomingDomainTypeSIPDirect()`, add the `"ai"` case before `default` (after line 100):

```go
case "ai":
    return h.startIncomingDomainTypeSIPDirectAI(ctx, cn, d, source)
```

**Step 3: Add the handler function**

Add after `startIncomingDomainTypeSIPDirectConference()` (after line 196):

```go
// startIncomingDomainTypeSIPDirectAI handles direct hash call routed to an AI resource.
func (h *callHandler) startIncomingDomainTypeSIPDirectAI(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectAI",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	a, err := h.reqHandler.AIV1AIGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get AI. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("ai", a).Debugf("Retrieved AI info. ai_id: %s", a.ID)

	destination := &commonaddress.Address{
		Type:   commonaddress.TypeAI,
		Target: d.ResourceID.String(),
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeAnswer,
		},
		{
			Type: fmaction.TypeAITalk,
			Option: fmaction.ConvertOption(fmaction.OptionAITalk{
				AssistanceType: amaicall.AssistanceTypeAI,
				AssistanceID:   a.ID,
			}),
		},
	}

	tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, a.CustomerID, fmflow.TypeFlow, "tmp", fmt.Sprintf("tmp flow for direct ai call. ai_id: %s", a.ID), actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, a.CustomerID, tmpFlow.ID, source, destination, nil)
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP_directAI$`
Expected: PASS

**Step 5: Add error-path tests**

Add `Test_startIncomingDomainTypeSIP_directAI_aiGetError` — verifies hangup with `ChannelCauseNoRouteDestination` when `AIV1AIGet` fails:

```go
func Test_startIncomingDomainTypeSIP_directAI_aiGetError(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			name: "ai get returns error",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.401",

				DestinationNumber: "direct.e90e5ce89f4d",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			})
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.e90e5ce89f4d").Return(&dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("27b03598-34df-46af-8b37-8d1ee0d7439d"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "ai",
				ResourceID:   uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
			}, nil)
			mockReq.EXPECT().AIV1AIGet(ctx, uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3")).Return(nil, fmt.Errorf("ai not found"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

Add `Test_startIncomingDomainTypeSIP_directAI_flowCreateError` — verifies hangup with `ChannelCauseNetworkOutOfOrder` when `FlowV1FlowCreate` fails:

```go
func Test_startIncomingDomainTypeSIP_directAI_flowCreateError(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			name: "flow create returns error",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.402",

				DestinationNumber: "direct.e90e5ce89f4d",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			responseAI := &amai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-ai",
			}

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			})
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.e90e5ce89f4d").Return(&dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("27b03598-34df-46af-8b37-8d1ee0d7439d"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "ai",
				ResourceID:   uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3"),
			}, nil)
			mockReq.EXPECT().AIV1AIGet(ctx, uuid.FromStringOrNil("2a8f731f-f34a-46a8-b4ca-e101797728f3")).Return(responseAI, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, responseAI.CustomerID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("flow create error"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNetworkOutOfOrder).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

**Step 6: Run all AI direct tests**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP_directAI`
Expected: ALL PASS (happy path + both error paths)

---

### Task 3: Add `"ai_team"` handler — test + implementation

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_test.go`
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go`

**Step 1: Write the failing test**

Add `Test_startIncomingDomainTypeSIP_directAITeam` to the test file. Same pattern as AI test but with `resource_type: "ai_team"` and `AIV1TeamGet`:

```go
func Test_startIncomingDomainTypeSIP_directAITeam(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource *commonaddress.Address
		responseDirect *dmdirect.Direct
		responseTeam   *amteam.Team
		responseFlow   *fmflow.Flow
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.410",

				DestinationNumber: "direct.b1c2d3e4f567",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a2a3a4-b5b6-c7c8-d9da-e1e2e3e4e5e6"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "ai_team",
				ResourceID:   uuid.FromStringOrNil("c1c2c3c4-d5d6-e7e8-f9fa-a1b2c3d4e5f6"),
				Hash:         "b1c2d3e4f567",
			},
			responseTeam: &amteam.Team{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1c2c3c4-d5d6-e7e8-f9fa-a1b2c3d4e5f6"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-team",
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1f2f3f4-f5f6-f7f8-f9fa-fbfcfdfeff02"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.b1c2d3e4f567").Return(tt.responseDirect, nil)
			mockReq.EXPECT().AIV1TeamGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseTeam, nil)

			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type: fmaction.TypeAITalk,
					Option: fmaction.ConvertOption(fmaction.OptionAITalk{
						AssistanceType: amaicall.AssistanceTypeTeam,
						AssistanceID:   tt.responseTeam.ID,
					}),
				},
			}

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseTeam.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				fmt.Sprintf("tmp flow for direct ai team call. team_id: %s", tt.responseTeam.ID),
				expectActions,
				uuid.Nil,
				false,
			).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseTeam.CustomerID).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseTeam.CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

New import needed in the test file:
```go
amteam "monorepo/bin-ai-manager/models/team"
```

**Step 2: Run test to verify it fails**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP_directAITeam`
Expected: FAIL

**Step 3: Add switch case and handler**

In `startIncomingDomainTypeSIPDirect()`, add before `default`:
```go
case "ai_team":
    return h.startIncomingDomainTypeSIPDirectAITeam(ctx, cn, d, source)
```

Add the handler function:

```go
// startIncomingDomainTypeSIPDirectAITeam handles direct hash call routed to an AI team resource.
func (h *callHandler) startIncomingDomainTypeSIPDirectAITeam(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectAITeam",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	team, err := h.reqHandler.AIV1TeamGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get AI team. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("team", team).Debugf("Retrieved AI team info. team_id: %s", team.ID)

	destination := &commonaddress.Address{
		Type:   commonaddress.TypeAI,
		Target: d.ResourceID.String(),
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeAnswer,
		},
		{
			Type: fmaction.TypeAITalk,
			Option: fmaction.ConvertOption(fmaction.OptionAITalk{
				AssistanceType: amaicall.AssistanceTypeTeam,
				AssistanceID:   team.ID,
			}),
		},
	}

	tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, team.CustomerID, fmflow.TypeFlow, "tmp", fmt.Sprintf("tmp flow for direct ai team call. team_id: %s", team.ID), actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, team.CustomerID, tmpFlow.ID, source, destination, nil)
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP_directAITeam$`
Expected: PASS

**Step 5: Add error-path tests**

Add `Test_startIncomingDomainTypeSIP_directAITeam_teamGetError`:

```go
func Test_startIncomingDomainTypeSIP_directAITeam_teamGetError(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			name: "team get returns error",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.411",

				DestinationNumber: "direct.b1c2d3e4f567",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			})
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.b1c2d3e4f567").Return(&dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a2a3a4-b5b6-c7c8-d9da-e1e2e3e4e5e6"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "ai_team",
				ResourceID:   uuid.FromStringOrNil("c1c2c3c4-d5d6-e7e8-f9fa-a1b2c3d4e5f6"),
			}, nil)
			mockReq.EXPECT().AIV1TeamGet(ctx, uuid.FromStringOrNil("c1c2c3c4-d5d6-e7e8-f9fa-a1b2c3d4e5f6")).Return(nil, fmt.Errorf("team not found"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

Add `Test_startIncomingDomainTypeSIP_directAITeam_flowCreateError`:

```go
func Test_startIncomingDomainTypeSIP_directAITeam_flowCreateError(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			name: "flow create returns error",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.412",

				DestinationNumber: "direct.b1c2d3e4f567",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			responseTeam := &amteam.Team{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1c2c3c4-d5d6-e7e8-f9fa-a1b2c3d4e5f6"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-team",
			}

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			})
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.b1c2d3e4f567").Return(&dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a1a2a3a4-b5b6-c7c8-d9da-e1e2e3e4e5e6"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "ai_team",
				ResourceID:   uuid.FromStringOrNil("c1c2c3c4-d5d6-e7e8-f9fa-a1b2c3d4e5f6"),
			}, nil)
			mockReq.EXPECT().AIV1TeamGet(ctx, uuid.FromStringOrNil("c1c2c3c4-d5d6-e7e8-f9fa-a1b2c3d4e5f6")).Return(responseTeam, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, responseTeam.CustomerID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("flow create error"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNetworkOutOfOrder).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

**Step 6: Run all AI Team direct tests**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP_directAITeam`
Expected: ALL PASS (happy path + both error paths)

---

### Task 4: Add `"agent"` handler — test + implementation

**Files:**
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_test.go`
- Modify: `bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go`

**Step 1: Write the failing test**

Add `Test_startIncomingDomainTypeSIP_directAgent`. Agent uses `TypeConnect` pattern (like extension):

```go
func Test_startIncomingDomainTypeSIP_directAgent(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource *commonaddress.Address
		responseDirect *dmdirect.Direct
		responseAgent  *amagent.Agent
		responseFlow   *fmflow.Flow
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.420",

				DestinationNumber: "direct.f1e2d3c4b5a6",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},
			responseDirect: &dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1b2b3b4-c5c6-d7d8-e9ea-f1f2f3f4f5f6"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "agent",
				ResourceID:   uuid.FromStringOrNil("d1d2d3d4-e5e6-f7f8-a9ba-b1c2d3e4f506"),
				Hash:         "f1e2d3c4b5a6",
			},
			responseAgent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1d2d3d4-e5e6-f7f8-a9ba-b1c2d3e4f506"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-agent",
			},
			responseFlow: &fmflow.Flow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f1f2f3f4-f5f6-f7f8-f9fa-fbfcfdfeff03"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(tt.responseSource)
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.f1e2d3c4b5a6").Return(tt.responseDirect, nil)
			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.responseDirect.ResourceID).Return(tt.responseAgent, nil)

			expectDestination := commonaddress.Address{
				Type:       commonaddress.TypeAgent,
				Target:     tt.responseAgent.ID.String(),
				TargetName: tt.responseAgent.Name,
			}
			expectActions := []fmaction.Action{
				{
					Type: fmaction.TypeConnect,
					Option: fmaction.ConvertOption(fmaction.OptionConnect{
						Source:       *tt.responseSource,
						Destinations: []commonaddress.Address{expectDestination},
						EarlyMedia:   false,
						RelayReason:  false,
					}),
				},
			}

			mockReq.EXPECT().FlowV1FlowCreate(
				ctx,
				tt.responseAgent.CustomerID,
				fmflow.TypeFlow,
				"tmp",
				fmt.Sprintf("tmp flow for direct agent call. agent_id: %s", tt.responseAgent.ID),
				expectActions,
				uuid.Nil,
				false,
			).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockReq.EXPECT().CustomerV1CustomerGet(ctx, tt.responseAgent.CustomerID).Return(&cucustomer.Customer{Status: cucustomer.StatusActive}, nil)
			mockReq.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, tt.responseAgent.CustomerID, bmbilling.ReferenceTypeCall, gomock.Any(), 1).Return(true, nil)
			mockUtil.EXPECT().UUIDCreate().Return(utilhandler.UUIDCreate())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

New import needed in the test file:
```go
amagent "monorepo/bin-agent-manager/models/agent"
```

**Step 2: Run test to verify it fails**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP_directAgent`
Expected: FAIL

**Step 3: Add switch case and handler**

In `startIncomingDomainTypeSIPDirect()`, add before `default`:
```go
case "agent":
    return h.startIncomingDomainTypeSIPDirectAgent(ctx, cn, d, source)
```

Add the handler function:

```go
// startIncomingDomainTypeSIPDirectAgent handles direct hash call routed to an agent resource.
func (h *callHandler) startIncomingDomainTypeSIPDirectAgent(ctx context.Context, cn *channel.Channel, d *dmdirect.Direct, source *commonaddress.Address) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "startIncomingDomainTypeSIPDirectAgent",
		"channel_id":  cn.ID,
		"resource_id": d.ResourceID,
	})

	ag, err := h.reqHandler.AgentV1AgentGet(ctx, d.ResourceID)
	if err != nil {
		log.Errorf("Could not get agent. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNoRouteDestination)
		return nil
	}
	log.WithField("agent", ag).Debugf("Retrieved agent info. agent_id: %s", ag.ID)

	destination := &commonaddress.Address{
		Type:       commonaddress.TypeAgent,
		Target:     ag.ID.String(),
		TargetName: ag.Name,
	}

	actions := []fmaction.Action{
		{
			Type: fmaction.TypeConnect,
			Option: fmaction.ConvertOption(fmaction.OptionConnect{
				Source:       *source,
				Destinations: []commonaddress.Address{*destination},
				EarlyMedia:   false,
				RelayReason:  false,
			}),
		},
	}

	tmpFlow, err := h.reqHandler.FlowV1FlowCreate(ctx, ag.CustomerID, fmflow.TypeFlow, "tmp", fmt.Sprintf("tmp flow for direct agent call. agent_id: %s", ag.ID), actions, uuid.Nil, false)
	if err != nil {
		log.Errorf("Could not create flow. err: %v", err)
		_, _ = h.channelHandler.HangingUp(ctx, cn.ID, ari.ChannelCauseNetworkOutOfOrder)
		return nil
	}

	h.startCallTypeFlow(ctx, cn, ag.CustomerID, tmpFlow.ID, source, destination, nil)
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP_directAgent$`
Expected: PASS

**Step 5: Add error-path tests**

Add `Test_startIncomingDomainTypeSIP_directAgent_agentGetError`:

```go
func Test_startIncomingDomainTypeSIP_directAgent_agentGetError(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			name: "agent get returns error",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.421",

				DestinationNumber: "direct.f1e2d3c4b5a6",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			})
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.f1e2d3c4b5a6").Return(&dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1b2b3b4-c5c6-d7d8-e9ea-f1f2f3f4f5f6"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "agent",
				ResourceID:   uuid.FromStringOrNil("d1d2d3d4-e5e6-f7f8-a9ba-b1c2d3e4f506"),
			}, nil)
			mockReq.EXPECT().AgentV1AgentGet(ctx, uuid.FromStringOrNil("d1d2d3d4-e5e6-f7f8-a9ba-b1c2d3e4f506")).Return(nil, fmt.Errorf("agent not found"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNoRouteDestination).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

Add `Test_startIncomingDomainTypeSIP_directAgent_flowCreateError`:

```go
func Test_startIncomingDomainTypeSIP_directAgent_flowCreateError(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel
	}{
		{
			name: "flow create returns error",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.422",

				DestinationNumber: "direct.f1e2d3c4b5a6",
				SourceNumber:      "+821100000002",

				StasisData: map[channel.StasisDataType]string{
					"context": "call-in",
					"domain":  "sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				reqHandler:     mockReq,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			responseAgent := &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1d2d3d4-e5e6-f7f8-a9ba-b1c2d3e4f506"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Name: "test-agent",
			}

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeTel).Return(&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			})
			mockReq.EXPECT().DirectV1DirectGetByHash(ctx, "direct.f1e2d3c4b5a6").Return(&dmdirect.Direct{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1b2b3b4-c5c6-d7d8-e9ea-f1f2f3f4f5f6"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				ResourceType: "agent",
				ResourceID:   uuid.FromStringOrNil("d1d2d3d4-e5e6-f7f8-a9ba-b1c2d3e4f506"),
			}, nil)
			mockReq.EXPECT().AgentV1AgentGet(ctx, uuid.FromStringOrNil("d1d2d3d4-e5e6-f7f8-a9ba-b1c2d3e4f506")).Return(responseAgent, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, responseAgent.CustomerID, fmflow.TypeFlow, "tmp", gomock.Any(), gomock.Any(), uuid.Nil, false).Return(nil, fmt.Errorf("flow create error"))
			mockChannel.EXPECT().HangingUp(ctx, tt.channel.ID, ari.ChannelCauseNetworkOutOfOrder).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
```

**Step 6: Run all Agent direct tests**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP_directAgent`
Expected: ALL PASS (happy path + both error paths)

---

### Task 5: Run full verification and commit

**Step 1: Run all tests in the package**

Run: `cd bin-call-manager && go test -v ./pkg/callhandler/ -run Test_startIncomingDomainTypeSIP`
Expected: ALL PASS (existing extension/conference tests still pass, new ai/ai_team/agent tests pass)

**Step 2: Run full verification workflow for both modified services**

Run bin-common-handler verification:
```bash
cd bin-common-handler && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Run bin-call-manager verification:
```bash
cd bin-call-manager && \
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```
Expected: All steps pass for both services with no errors.

**Step 3: Commit**

```bash
git add bin-common-handler/models/address/main.go \
        bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip.go \
        bin-call-manager/pkg/callhandler/start_incoming_domain_type_sip_test.go \
        bin-call-manager/go.mod bin-call-manager/go.sum \
        docs/plans/2026-03-26-add-direct-sip-handlers-ai-agent.md
git commit -m "NOJIRA-Add-direct-sip-handlers-for-ai-agent-resource-types

Add missing SIP direct call handlers for ai, ai_team, and agent resource
types in call-manager. Previously, only extension and conference were handled,
causing SIP 404 for direct calls to AI resources.

- bin-common-handler: Add TypeAI address type for AI resource destinations
- bin-call-manager: Add startIncomingDomainTypeSIPDirectAI handler for ai resource type
- bin-call-manager: Add startIncomingDomainTypeSIPDirectAITeam handler for ai_team resource type
- bin-call-manager: Add startIncomingDomainTypeSIPDirectAgent handler for agent resource type
- bin-call-manager: Add tests for all three new direct call handlers (happy path, resource get error, flow create error)
- docs: Add design document for the fix"
```

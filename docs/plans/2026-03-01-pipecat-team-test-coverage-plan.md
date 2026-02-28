# Pipecat Manager Team Test Coverage Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close Phase 1 team support test coverage gaps in bin-pipecat-manager by adding tests for `runGetLLMKey()` and `getToolsForPipecatcall()`.

**Architecture:** Both functions live in `bin-pipecat-manager/pkg/pipecatcallhandler/`. They resolve the AI from an AIcall (which may be team-backed) via `resolveAIFromAIcall()`, then extract the LLM key or filter tools respectively. Tests use gomock with `requesthandler.MockRequestHandler` and `toolhandler.MockToolHandler`.

**Tech Stack:** Go, gomock (go.uber.org/mock), table-driven tests

---

### Task 1: Add tests for runGetLLMKey()

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/run_test.go` (append after existing tests)

**Step 1: Write the test**

Add a table-driven test `Test_runGetLLMKey` to `run_test.go` covering:

1. **AICall reference with AI assistance** - resolves AI directly, returns EngineKey
2. **AICall reference with team assistance** - fetches team, finds start member, resolves AI, returns EngineKey
3. **AICall reference with AIcall fetch error** - returns empty string
4. **AICall reference with AI resolve error** - returns empty string
5. **Non-AICall reference type** - returns empty string (default case)

```go
func Test_runGetLLMKey(t *testing.T) {
	aiID := uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111")
	teamID := uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222")
	memberID := uuid.FromStringOrNil("c3c3c3c3-3333-3333-3333-333333333333")
	memberAIID := uuid.FromStringOrNil("d4d4d4d4-4444-4444-4444-444444444444")
	pipecatcallID := uuid.FromStringOrNil("e5e5e5e5-5555-5555-5555-555555555555")
	referenceID := uuid.FromStringOrNil("f6f6f6f6-6666-6666-6666-666666666666")

	tests := []struct {
		name string

		pc *pipecatcall.Pipecatcall

		prepareMockFn func(mockReq *requesthandler.MockRequestHandler)

		expectedKey string
	}{
		{
			name: "aicall reference with ai assistance returns key",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeAI,
					AssistanceID:   aiID,
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: aiID,
					},
					EngineKey: "ai-direct-key",
				}, nil)
			},

			expectedKey: "ai-direct-key",
		},
		{
			name: "aicall reference with team assistance returns start member key",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeTeam,
					AssistanceID:   teamID,
				}, nil)
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amateam.Team{
					Identity: commonidentity.Identity{
						ID: teamID,
					},
					StartMemberID: memberID,
					Members: []amateam.Member{
						{ID: memberID, AIID: memberAIID},
					},
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), memberAIID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: memberAIID,
					},
					EngineKey: "team-member-key",
				}, nil)
			},

			expectedKey: "team-member-key",
		},
		{
			name: "aicall reference with aicall fetch error returns empty",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("aicall not found"))
			},

			expectedKey: "",
		},
		{
			name: "aicall reference with ai resolve error returns empty",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeAI,
					AssistanceID:   aiID,
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(nil, fmt.Errorf("ai not found"))
			},

			expectedKey: "",
		},
		{
			name: "non-aicall reference type returns empty",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeCall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler) {
				// no RPC calls expected for non-AICall reference
			},

			expectedKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			tt.prepareMockFn(mockReq)

			h := &pipecatcallHandler{
				requestHandler: mockReq,
			}

			result := h.runGetLLMKey(context.Background(), tt.pc)
			if result != tt.expectedKey {
				t.Errorf("Wrong LLM key. expect: %q, got: %q", tt.expectedKey, result)
			}
		})
	}
}
```

**Step 2: Run test to verify it passes**

Run: `cd bin-pipecat-manager && go test -v -run Test_runGetLLMKey ./pkg/pipecatcallhandler/...`
Expected: PASS (all 5 subtests pass)

**Step 3: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/run_test.go
git commit -m "NOJIRA-add-pipecat-team-tests

- bin-pipecat-manager: Add Test_runGetLLMKey covering AI, team, error, and default paths"
```

---

### Task 2: Add tests for getToolsForPipecatcall()

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner_test.go` (append after existing tests)

**Step 1: Write the test**

Add a table-driven test `Test_getToolsForPipecatcall` to `runner_test.go` covering:

1. **AICall with AI assistance** - filters tools by AI's ToolNames
2. **AICall with team assistance** - resolves start member's AI, filters by ToolNames
3. **Non-AICall reference** - returns all tools
4. **AICall with aicall fetch error** - falls back to all tools
5. **AICall with AI resolve error** - falls back to all tools

```go
func Test_getToolsForPipecatcall(t *testing.T) {
	aiID := uuid.FromStringOrNil("a1a1a1a1-1111-1111-1111-111111111111")
	teamID := uuid.FromStringOrNil("b2b2b2b2-2222-2222-2222-222222222222")
	memberID := uuid.FromStringOrNil("c3c3c3c3-3333-3333-3333-333333333333")
	memberAIID := uuid.FromStringOrNil("d4d4d4d4-4444-4444-4444-444444444444")
	pipecatcallID := uuid.FromStringOrNil("e5e5e5e5-5555-5555-5555-555555555555")
	referenceID := uuid.FromStringOrNil("f6f6f6f6-6666-6666-6666-666666666666")

	allTools := []aitool.Tool{
		{Name: aitool.ToolNameConnectCall, Description: "connect call"},
		{Name: aitool.ToolNameSendEmail, Description: "send email"},
		{Name: aitool.ToolNameStopFlow, Description: "stop flow"},
	}

	filteredTools := []aitool.Tool{
		{Name: aitool.ToolNameConnectCall, Description: "connect call"},
		{Name: aitool.ToolNameStopFlow, Description: "stop flow"},
	}

	tests := []struct {
		name string

		pc *pipecatcall.Pipecatcall

		prepareMockFn func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler)

		expectedTools []aitool.Tool
	}{
		{
			name: "aicall with ai assistance filters tools by ai tool names",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeAI,
					AssistanceID:   aiID,
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: aiID,
					},
					ToolNames: []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameStopFlow},
				}, nil)
				mockTool.EXPECT().GetByNames([]aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameStopFlow}).Return(filteredTools)
			},

			expectedTools: filteredTools,
		},
		{
			name: "aicall with team assistance filters tools by start member ai tool names",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeTeam,
					AssistanceID:   teamID,
				}, nil)
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(&amateam.Team{
					Identity: commonidentity.Identity{
						ID: teamID,
					},
					StartMemberID: memberID,
					Members: []amateam.Member{
						{ID: memberID, AIID: memberAIID},
					},
				}, nil)
				mockReq.EXPECT().AIV1AIGet(gomock.Any(), memberAIID).Return(&amai.AI{
					Identity: commonidentity.Identity{
						ID: memberAIID,
					},
					ToolNames: []aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameStopFlow},
				}, nil)
				mockTool.EXPECT().GetByNames([]aitool.ToolName{aitool.ToolNameConnectCall, aitool.ToolNameStopFlow}).Return(filteredTools)
			},

			expectedTools: filteredTools,
		},
		{
			name: "non-aicall reference returns all tools",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeCall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockTool.EXPECT().GetAll().Return(allTools)
			},

			expectedTools: allTools,
		},
		{
			name: "aicall fetch error falls back to all tools",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(nil, fmt.Errorf("aicall not found"))
				mockTool.EXPECT().GetAll().Return(allTools)
			},

			expectedTools: allTools,
		},
		{
			name: "ai resolve error falls back to all tools",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: pipecatcallID,
				},
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   referenceID,
			},

			prepareMockFn: func(mockReq *requesthandler.MockRequestHandler, mockTool *toolhandler.MockToolHandler) {
				mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), referenceID).Return(&amaicall.AIcall{
					AssistanceType: amaicall.AssistanceTypeTeam,
					AssistanceID:   teamID,
				}, nil)
				mockReq.EXPECT().AIV1TeamGet(gomock.Any(), teamID).Return(nil, fmt.Errorf("team not found"))
				mockTool.EXPECT().GetAll().Return(allTools)
			},

			expectedTools: allTools,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockTool := toolhandler.NewMockToolHandler(mc)
			tt.prepareMockFn(mockReq, mockTool)

			h := &pipecatcallHandler{
				requestHandler: mockReq,
				toolHandler:    mockTool,
			}

			result := h.getToolsForPipecatcall(context.Background(), tt.pc)
			if len(result) != len(tt.expectedTools) {
				t.Errorf("Wrong number of tools. expect: %d, got: %d", len(tt.expectedTools), len(result))
				return
			}

			for i, tool := range result {
				if tool.Name != tt.expectedTools[i].Name {
					t.Errorf("Wrong tool at index %d. expect: %s, got: %s", i, tt.expectedTools[i].Name, tool.Name)
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it passes**

Run: `cd bin-pipecat-manager && go test -v -run Test_getToolsForPipecatcall ./pkg/pipecatcallhandler/...`
Expected: PASS (all 5 subtests pass)

**Step 3: Commit**

```bash
git add bin-pipecat-manager/pkg/pipecatcallhandler/runner_test.go
git commit -m "NOJIRA-add-pipecat-team-tests

- bin-pipecat-manager: Add Test_getToolsForPipecatcall covering AI, team, fallback, and error paths"
```

---

### Task 3: Run full verification workflow

**Step 1: Run the complete verification workflow**

```bash
cd bin-pipecat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All steps pass with no errors.

**Step 2: Final commit (if any vendor/generated changes)**

If `go mod tidy` or `go generate` produces changes, commit them:

```bash
git add -A
git commit -m "NOJIRA-add-pipecat-team-tests

- bin-pipecat-manager: Run verification workflow (tidy, vendor, generate)"
```

---

### Notes

- The tests use the existing mock infrastructure (`requesthandler.MockRequestHandler`, `toolhandler.MockToolHandler`)
- Import `amateam "monorepo/bin-ai-manager/models/team"` is already used in `run_test.go`
- Import `aitool "monorepo/bin-ai-manager/models/tool"` and `"monorepo/bin-pipecat-manager/pkg/toolhandler"` need to be added to `runner_test.go`
- Both test functions follow the established table-driven pattern with `prepareMockFn`

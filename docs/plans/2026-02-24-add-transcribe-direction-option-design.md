# Add Direction Option for transcribe_start and transcribe_recording Actions

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Allow users to specify audio direction (in/out/both) for transcribe_start and transcribe_recording flow actions, instead of hardcoding "both".

**Architecture:** Add `Direction` field to flow action option structs, read it in action handlers with default fallback to "both", update OpenAPI schemas and RST docs. The transcribe model, transcribe-manager, and API layer already support direction — this only wires the flow action layer.

**Tech Stack:** Go, OpenAPI 3.0 (oapi-codegen), RST/Sphinx docs

**Worktree:** `~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option`

---

### Task 1: Add Direction field to flow action option structs

**Files:**
- Modify: `bin-flow-manager/models/action/option.go:303-320`

**Step 1: Add Direction field to OptionTranscribeStart**

In `bin-flow-manager/models/action/option.go`, change the `OptionTranscribeStart` struct (line 304-308) from:

```go
// OptionTranscribeStart defines action TypeTranscribeStart's option.
type OptionTranscribeStart struct {
	Language    string    `json:"language,omitempty"`       // BCP47 format. en-US
	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
	Provider    string    `json:"provider,omitempty"`       // transcribe provider(gcp/aws)
}
```

To:

```go
// OptionTranscribeStart defines action TypeTranscribeStart's option.
type OptionTranscribeStart struct {
	Language    string    `json:"language,omitempty"`       // BCP47 format. en-US
	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
	Provider    string    `json:"provider,omitempty"`       // transcribe provider(gcp/aws)
	Direction   string    `json:"direction,omitempty"`      // in|out|both. default: both
}
```

**Step 2: Add Direction field to OptionTranscribeRecording**

In the same file, change the `OptionTranscribeRecording` struct (line 316-320) from:

```go
// OptionTranscribeRecording defines action TypeTranscribeRecording's option.
type OptionTranscribeRecording struct {
	Language    string    `json:"language,omitempty"`       // BCP47 format. en-US
	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
	Provider    string    `json:"provider,omitempty"`       // transcribe provider(gcp/aws)
}
```

To:

```go
// OptionTranscribeRecording defines action TypeTranscribeRecording's option.
type OptionTranscribeRecording struct {
	Language    string    `json:"language,omitempty"`       // BCP47 format. en-US
	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // flow id for the end of recording.
	Provider    string    `json:"provider,omitempty"`       // transcribe provider(gcp/aws)
	Direction   string    `json:"direction,omitempty"`      // in|out|both. default: both
}
```

**Step 3: Update option tests**

In `bin-flow-manager/models/action/option_test.go`, update `Test_OptionTranscribeStart` (line 1032). Add a test case with direction:

```go
{
    name: "with direction",

    option: []byte(`{"language": "en-US", "on_end_flow_id": "ccde4a38-093b-11f0-921a-93245e27ef98", "direction": "in"}`),

    expectedRes: OptionTranscribeStart{
        Language:    "en-US",
        OnEndFlowID: uuid.FromStringOrNil("ccde4a38-093b-11f0-921a-93245e27ef98"),
        Direction:   "in",
    },
},
```

Update `Test_OptionTranscribeRecording` (line 1069). Add a test case with direction:

```go
{
    name: "with direction",

    option: []byte(`{"language": "en-US", "on_end_flow_id": "cd02b6a2-093b-11f0-b71b-7bc0ff6efaaf", "direction": "out"}`),

    expectedRes: OptionTranscribeRecording{
        Language:    "en-US",
        OnEndFlowID: uuid.FromStringOrNil("cd02b6a2-093b-11f0-b71b-7bc0ff6efaaf"),
        Direction:   "out",
    },
},
```

**Step 4: Run tests**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option/bin-flow-manager
go test ./models/action/... -run "Test_OptionTranscribeStart|Test_OptionTranscribeRecording" -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add bin-flow-manager/models/action/option.go bin-flow-manager/models/action/option_test.go
git commit -m "NOJIRA-add-transcribe-direction-option

- bin-flow-manager: Add Direction field to OptionTranscribeStart and OptionTranscribeRecording structs"
```

---

### Task 2: Wire direction through action handlers

**Files:**
- Modify: `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:637-737`
- Modify: `bin-flow-manager/pkg/activeflowhandler/actionhandle_test.go:3347-3515`

**Step 1: Update actionHandleTranscribeStart**

In `bin-flow-manager/pkg/activeflowhandler/actionhandle.go`, in the `actionHandleTranscribeStart` function (line 691-737), replace line 726:

```go
		tmtranscribe.DirectionBoth,
```

With:

```go
		tmtranscribe.Direction(opt.Direction),
```

And add the default logic after the unmarshal block (after line 715), before the `// transcribe start` comment:

```go
	if opt.Direction == "" {
		opt.Direction = string(tmtranscribe.DirectionBoth)
	}
```

**Step 2: Update actionHandleTranscribeRecording**

In the same file, in the `actionHandleTranscribeRecording` function (line 637-689), replace line 678:

```go
			tmtranscribe.DirectionBoth,
```

With:

```go
			tmtranscribe.Direction(opt.Direction),
```

And add the default logic after the unmarshal block (after line 655), before the reference type check:

```go
	if opt.Direction == "" {
		opt.Direction = string(tmtranscribe.DirectionBoth)
	}
```

**Step 3: Update Test_actionHandleTranscribeStart**

In `bin-flow-manager/pkg/activeflowhandler/actionhandle_test.go`, update the test (line 3429-3515):

- Add `expectedDirection tmtranscribe.Direction` to the test struct (after line 3440)
- In the "normal" test case, add `"direction": "in"` to the Option map (line 3458-3461) and set `expectedDirection: tmtranscribe.DirectionIn`
- Add a second test case "default direction" WITHOUT direction in the Option map, with `expectedDirection: tmtranscribe.DirectionBoth`
- In the mock expectation (line 3506), change `tmtranscribe.DirectionBoth` to `tt.expectedDirection`

Updated test struct and cases:

```go
type test struct {
    name       string
    activeFlow *activeflow.Activeflow

    expectedCustomerID    uuid.UUID
    expectedActiveflowID  uuid.UUID
    expectedOnEndFlowID   uuid.UUID
    expectedReferenceID   uuid.UUID
    expectedReferenceType tmtranscribe.ReferenceType
    expectedLanguage      string
    expectedDirection     tmtranscribe.Direction

    response *tmtranscribe.Transcribe
}

tests := []test{
    {
        name: "with direction",
        activeFlow: &activeflow.Activeflow{
            Identity: commonidentity.Identity{
                ID:         uuid.FromStringOrNil("cfd0865a-093d-11f0-bdc8-87ff6a57d585"),
                CustomerID: uuid.FromStringOrNil("b4d3fb66-8795-11ec-997c-7f2786edbef2"),
            },
            ReferenceType: activeflow.ReferenceTypeCall,
            ReferenceID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
            CurrentAction: action.Action{
                ID:   uuid.FromStringOrNil("0737bd5c-0c08-11ec-9ba8-3bc700c21fd4"),
                Type: action.TypeTranscribeStart,
                Option: map[string]any{
                    "language":       "en-US",
                    "on_end_flow_id": "bf629ef8-093c-11f0-a38e-73d3a32d02a6",
                    "direction":      "in",
                },
            },
        },

        expectedCustomerID:    uuid.FromStringOrNil("b4d3fb66-8795-11ec-997c-7f2786edbef2"),
        expectedActiveflowID:  uuid.FromStringOrNil("cfd0865a-093d-11f0-bdc8-87ff6a57d585"),
        expectedOnEndFlowID:   uuid.FromStringOrNil("bf629ef8-093c-11f0-a38e-73d3a32d02a6"),
        expectedReferenceID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
        expectedReferenceType: tmtranscribe.ReferenceTypeCall,
        expectedLanguage:      "en-US",
        expectedDirection:     tmtranscribe.DirectionIn,

        response: &tmtranscribe.Transcribe{
            Identity: commonidentity.Identity{
                ID: uuid.FromStringOrNil("e1e69720-0c08-11ec-9f5c-db1f63f63215"),
            },
            ReferenceType: tmtranscribe.ReferenceTypeCall,
            ReferenceID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
            HostID:        uuid.FromStringOrNil("f91b4f58-0c08-11ec-88fd-cfbbb1957a54"),
            Language:      "en-US",
        },
    },
    {
        name: "default direction",
        activeFlow: &activeflow.Activeflow{
            Identity: commonidentity.Identity{
                ID:         uuid.FromStringOrNil("cfd0865a-093d-11f0-bdc8-87ff6a57d585"),
                CustomerID: uuid.FromStringOrNil("b4d3fb66-8795-11ec-997c-7f2786edbef2"),
            },
            ReferenceType: activeflow.ReferenceTypeCall,
            ReferenceID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
            CurrentAction: action.Action{
                ID:   uuid.FromStringOrNil("0737bd5c-0c08-11ec-9ba8-3bc700c21fd4"),
                Type: action.TypeTranscribeStart,
                Option: map[string]any{
                    "language":       "en-US",
                    "on_end_flow_id": "bf629ef8-093c-11f0-a38e-73d3a32d02a6",
                },
            },
        },

        expectedCustomerID:    uuid.FromStringOrNil("b4d3fb66-8795-11ec-997c-7f2786edbef2"),
        expectedActiveflowID:  uuid.FromStringOrNil("cfd0865a-093d-11f0-bdc8-87ff6a57d585"),
        expectedOnEndFlowID:   uuid.FromStringOrNil("bf629ef8-093c-11f0-a38e-73d3a32d02a6"),
        expectedReferenceID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
        expectedReferenceType: tmtranscribe.ReferenceTypeCall,
        expectedLanguage:      "en-US",
        expectedDirection:     tmtranscribe.DirectionBoth,

        response: &tmtranscribe.Transcribe{
            Identity: commonidentity.Identity{
                ID: uuid.FromStringOrNil("e1e69720-0c08-11ec-9f5c-db1f63f63215"),
            },
            ReferenceType: tmtranscribe.ReferenceTypeCall,
            ReferenceID:   uuid.FromStringOrNil("01f28ffc-0c08-11ec-8b28-0f1dd70b3428"),
            HostID:        uuid.FromStringOrNil("f91b4f58-0c08-11ec-88fd-cfbbb1957a54"),
            Language:      "en-US",
        },
    },
}
```

In the mock expectation, change:
```go
tmtranscribe.DirectionBoth,
```
To:
```go
tt.expectedDirection,
```

**Step 4: Update Test_actionHandleTranscribeRecording**

In the same test file (line 3347-3427):

- Add `expectedDirection tmtranscribe.Direction` to the test struct
- In the "normal" test case, add `"direction": "out"` to the Option map and set `expectedDirection: tmtranscribe.DirectionOut`
- Add a second test case "default direction" without direction in Option, with `expectedDirection: tmtranscribe.DirectionBoth`
- In the mock expectation (line 3416), change `tmtranscribe.DirectionBoth` to `tt.expectedDirection`

**Step 5: Run tests**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option/bin-flow-manager
go test ./pkg/activeflowhandler/... -run "Test_actionHandleTranscribeStart|Test_actionHandleTranscribeRecording" -v
```

Expected: PASS

**Step 6: Commit**

```bash
git add bin-flow-manager/pkg/activeflowhandler/actionhandle.go bin-flow-manager/pkg/activeflowhandler/actionhandle_test.go
git commit -m "NOJIRA-add-transcribe-direction-option

- bin-flow-manager: Wire direction option through actionHandleTranscribeStart and actionHandleTranscribeRecording
- bin-flow-manager: Default to DirectionBoth when direction is empty"
```

---

### Task 3: Update OpenAPI schemas and regenerate

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml:3722-3752`
- Regenerate: `bin-openapi-manager/gens/models/gen.go`
- Regenerate: `bin-api-manager/gens/openapi_server/gen.go`

**Step 1: Update FlowManagerActionOptionTranscribeStart schema**

In `bin-openapi-manager/openapi/openapi.yaml`, change the `FlowManagerActionOptionTranscribeStart` schema (lines 3722-3734) from:

```yaml
    FlowManagerActionOptionTranscribeStart:
      type: object
      properties:
        language:
          type: string
          description: BCP47 format for the language (e.g., en-US).
          example: "en-US"
        on_end_flow_id:
          type: string
          format: uuid
          x-go-type: string
          description: The unique identifier of the flow to execute when transcription ends. Returned from the `POST /flows` or `GET /flows` response.
          example: "a1b2c3d4-e5f6-7890-1234-567890abcdef"
```

To:

```yaml
    FlowManagerActionOptionTranscribeStart:
      type: object
      properties:
        language:
          type: string
          description: BCP47 format for the language (e.g., en-US).
          example: "en-US"
        on_end_flow_id:
          type: string
          format: uuid
          x-go-type: string
          description: The unique identifier of the flow to execute when transcription ends. Returned from the `POST /flows` or `GET /flows` response.
          example: "a1b2c3d4-e5f6-7890-1234-567890abcdef"
        provider:
          $ref: '#/components/schemas/TranscribeManagerTranscribeProvider'
          description: "STT provider to use. If omitted, VoIPBIN selects the best available provider automatically."
          example: "gcp"
        direction:
          $ref: '#/components/schemas/TranscribeManagerTranscribeDirection'
          description: "Audio direction to transcribe. If omitted, defaults to both."
          example: "both"
```

**Step 2: Update FlowManagerActionOptionTranscribeRecording schema**

In the same file, change the `FlowManagerActionOptionTranscribeRecording` schema (lines 3740-3752) from:

```yaml
    FlowManagerActionOptionTranscribeRecording:
      type: object
      properties:
        language:
          type: string
          description: BCP47 format for the language (e.g., en-US).
          example: "en-US"
        on_end_flow_id:
          type: string
          format: uuid
          x-go-type: string
          description: The unique identifier of the flow to execute when recording transcription ends. Returned from the `POST /flows` or `GET /flows` response.
          example: "a1b2c3d4-e5f6-7890-1234-567890abcdef"
```

To:

```yaml
    FlowManagerActionOptionTranscribeRecording:
      type: object
      properties:
        language:
          type: string
          description: BCP47 format for the language (e.g., en-US).
          example: "en-US"
        on_end_flow_id:
          type: string
          format: uuid
          x-go-type: string
          description: The unique identifier of the flow to execute when recording transcription ends. Returned from the `POST /flows` or `GET /flows` response.
          example: "a1b2c3d4-e5f6-7890-1234-567890abcdef"
        provider:
          $ref: '#/components/schemas/TranscribeManagerTranscribeProvider'
          description: "STT provider to use. If omitted, VoIPBIN selects the best available provider automatically."
          example: "gcp"
        direction:
          $ref: '#/components/schemas/TranscribeManagerTranscribeDirection'
          description: "Audio direction to transcribe. If omitted, defaults to both."
          example: "both"
```

**Step 3: Regenerate bin-openapi-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option/bin-openapi-manager
go generate ./...
```

Expected: `gens/models/gen.go` updated with new fields.

**Step 4: Regenerate bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option/bin-api-manager
go generate ./...
```

Expected: `gens/openapi_server/gen.go` updated.

**Step 5: Commit**

```bash
git add bin-openapi-manager/openapi/openapi.yaml bin-openapi-manager/gens/models/gen.go bin-api-manager/gens/openapi_server/gen.go
git commit -m "NOJIRA-add-transcribe-direction-option

- bin-openapi-manager: Add direction and provider fields to FlowManagerActionOptionTranscribeStart and FlowManagerActionOptionTranscribeRecording schemas
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 4: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/flow_struct_action.rst:1201-1265`

**Step 1: Update transcribe_recording section**

In `bin-api-manager/docsdev/source/flow_struct_action.rst`, update the transcribe_recording Parameters section (lines 1209-1220) from:

```rst
Parameters
++++++++++
.. code::

    {
        "type": "transcribe_recording",
        "option": {
            "language": "<string>",
            "provider": "<string>"
        }
    }

* ``language`` (String): Language in BCP47 format (e.g., ``en-US``).
* ``provider`` (String, optional): STT provider to use: ``gcp`` or ``aws``. If omitted, VoIPBIN selects the best available provider automatically.
```

To:

```rst
Parameters
++++++++++
.. code::

    {
        "type": "transcribe_recording",
        "option": {
            "language": "<string>",
            "provider": "<string>",
            "direction": "<string>"
        }
    }

* ``language`` (String): Language in BCP47 format (e.g., ``en-US``).
* ``provider`` (String, optional): STT provider to use: ``gcp`` or ``aws``. If omitted, VoIPBIN selects the best available provider automatically.
* ``direction`` (String, optional): Audio direction to transcribe: ``in``, ``out``, or ``both``. Defaults to ``both``.
```

Update the transcribe_recording Example section (lines 1224-1232) from:

```rst
Example
+++++++
.. code::

    {
        "type": "transcribe_recording",
        "option": {
            "language": "en-US",
            "provider": "gcp"
        }
    }
```

To:

```rst
Example
+++++++
.. code::

    {
        "type": "transcribe_recording",
        "option": {
            "language": "en-US",
            "provider": "gcp",
            "direction": "both"
        }
    }
```

**Step 2: Update transcribe_start section**

Update the transcribe_start Parameters section (lines 1240-1253) from:

```rst
Parameters
++++++++++
.. code::

    {
        "type": "transcribe_start",
        "option": {
            "language": "<string>",
            "provider": "<string>"
        }
    }

* ``language`` (String): Language in BCP47 format. Examples: ``en-US``, ``ko-KR``. The value may be a two-letter language code (e.g., ``en``) or language code with country/region (e.g., ``en-US``).
* ``provider`` (String, optional): STT provider to use: ``gcp`` or ``aws``. If omitted, VoIPBIN selects the best available provider automatically.
```

To:

```rst
Parameters
++++++++++
.. code::

    {
        "type": "transcribe_start",
        "option": {
            "language": "<string>",
            "provider": "<string>",
            "direction": "<string>"
        }
    }

* ``language`` (String): Language in BCP47 format. Examples: ``en-US``, ``ko-KR``. The value may be a two-letter language code (e.g., ``en``) or language code with country/region (e.g., ``en-US``).
* ``provider`` (String, optional): STT provider to use: ``gcp`` or ``aws``. If omitted, VoIPBIN selects the best available provider automatically.
* ``direction`` (String, optional): Audio direction to transcribe: ``in``, ``out``, or ``both``. Defaults to ``both``.
```

Update the transcribe_start Example section (lines 1255-1265) from:

```rst
Example
+++++++
.. code::

    {
        "type": "transcribe_start",
        "option": {
            "language": "en-US",
            "provider": "gcp"
        }
    }
```

To:

```rst
Example
+++++++
.. code::

    {
        "type": "transcribe_start",
        "option": {
            "language": "en-US",
            "provider": "gcp",
            "direction": "both"
        }
    }
```

**Step 3: Rebuild HTML docs**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option/bin-api-manager/docsdev
rm -rf build
python3 -m sphinx -M html source build
```

**Step 4: Commit**

```bash
git add bin-api-manager/docsdev/source/flow_struct_action.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-add-transcribe-direction-option

- bin-api-manager: Add direction parameter documentation for transcribe_start and transcribe_recording actions"
```

---

### Task 5: Run full verification workflow

**Step 1: Verify bin-flow-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option/bin-flow-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass.

**Step 2: Verify bin-openapi-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option/bin-openapi-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass.

**Step 3: Verify bin-api-manager**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option/bin-api-manager
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass.

**Step 4: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo-worktrees/NOJIRA-add-transcribe-direction-option
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
```

Expected: No conflicts.

**Step 5: Push and create PR**

```bash
git push -u origin NOJIRA-add-transcribe-direction-option
gh pr create --title "NOJIRA-add-transcribe-direction-option" --body "$(cat <<'EOF'
Add direction option (in/out/both) to transcribe_start and transcribe_recording
flow actions, replacing the hardcoded DirectionBoth. Defaults to both when omitted
for backward compatibility. Also adds the missing provider and direction fields to
the OpenAPI schemas for these action options.

- bin-flow-manager: Add Direction field to OptionTranscribeStart and OptionTranscribeRecording
- bin-flow-manager: Wire direction through actionHandleTranscribeStart and actionHandleTranscribeRecording with default fallback
- bin-openapi-manager: Add direction and provider fields to FlowManagerActionOptionTranscribeStart and FlowManagerActionOptionTranscribeRecording schemas
- bin-api-manager: Regenerate server code from updated OpenAPI spec
- bin-api-manager: Update RST docs for transcribe_start and transcribe_recording actions
EOF
)"
```

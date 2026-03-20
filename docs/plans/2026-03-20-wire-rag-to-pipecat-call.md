# Wire RAG to Pipecat Call — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enable AI assistants during live pipecat voice calls to search a customer's knowledge base (RAG) via a `search_knowledge` tool.

**Architecture:** Add `rag_id` to the AI model, add a `search_knowledge` tool to the tool infrastructure, and wire tool execution through AI Manager's existing `ToolHandle()` dispatch pattern. RAG Manager already has query RPC; we just need to add `Text` to query responses and plumb the new field through the 4-layer parameter chain.

**Tech Stack:** Go, RabbitMQ RPC, MySQL (Alembic migrations), PostgreSQL (pgvector), OpenAPI, Sphinx RST docs

---

### Task 1: Add Text field to RAG query response

**Files:**
- Modify: `bin-rag-manager/models/query/main.go:15-20`
- Modify: `bin-rag-manager/pkg/raghandler/query.go:49-54`

**Step 1: Add Text field to Source struct**

In `bin-rag-manager/models/query/main.go`, add `Text` to the `Source` struct:

```go
// Source represents a source reference in the query response
type Source struct {
	DocumentID     uuid.UUID `json:"document_id"`
	DocumentName   string    `json:"document_name"`
	SectionTitle   string    `json:"section_title"`
	RelevanceScore float64   `json:"relevance_score"`
	Text           string    `json:"text"`
}
```

**Step 2: Populate Text from chunk in query handler**

In `bin-rag-manager/pkg/raghandler/query.go`, add `Text: c.Text` to the source builder:

```go
sources[i] = query.Source{
	DocumentID:     c.DocumentID,
	DocumentName:   docName,
	SectionTitle:   c.SectionTitle,
	RelevanceScore: scores[i],
	Text:           c.Text,
}
```

The chunk model (`models/chunk/main.go:17`) already has `Text string` — no chunk changes needed.

**Step 3: Run verification**

Run:
```bash
cd bin-rag-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
Expected: All pass. This is a backward-compatible additive change.

**Step 4: Commit**

```bash
git add bin-rag-manager/
git commit -m "NOJIRA-Wire-rag-to-pipecat-call

- bin-rag-manager: Add Text field to query.Source so RAG responses include chunk text"
```

---

### Task 2: Create Alembic migration for rag_id column

**Files:**
- Create: `bin-dbscheme-manager/alembic/versions/<new>_add_rag_id_to_ai_ais.py`

**Step 1: Generate migration file**

```bash
cd bin-dbscheme-manager && alembic -c alembic.ini revision -m "add_rag_id_to_ai_ais"
```

**Step 2: Write migration SQL**

Edit the generated file:

```python
def upgrade():
    op.execute("ALTER TABLE ai_ais ADD COLUMN rag_id binary(16) AFTER engine_key")


def downgrade():
    op.execute("ALTER TABLE ai_ais DROP COLUMN rag_id")
```

`rag_id` is nullable (no DEFAULT needed). Existing rows get NULL, meaning no knowledge base configured.

**Step 3: Commit**

```bash
git add bin-dbscheme-manager/
git commit -m "NOJIRA-Wire-rag-to-pipecat-call

- bin-dbscheme-manager: Add rag_id column to ai_ais table after engine_key"
```

**IMPORTANT:** Do NOT run `alembic upgrade`. The migration will be applied by a human operator.

---

### Task 3: Add RagID to AI model structs

**Files:**
- Modify: `bin-ai-manager/models/ai/main.go:44-71` (AI struct)
- Modify: `bin-ai-manager/models/ai/field.go:7-34` (Field constants)
- Modify: `bin-ai-manager/models/ai/webhook.go:13-67` (WebhookMessage + ConvertWebhookMessage)

**Step 1: Add RagID to AI struct**

In `bin-ai-manager/models/ai/main.go`, add `RagID` after `EngineKey` (line 52):

```go
EngineKey   string         `json:"engine_key,omitempty" db:"engine_key"` // ai(llm) service api key
RagID       uuid.UUID      `json:"rag_id,omitempty" db:"rag_id,uuid"`

InitPrompt string `json:"init_prompt,omitempty" db:"init_prompt"`
```

Also add the `uuid` import — check if `github.com/gofrs/uuid` is already imported. It is NOT currently imported in this file, so add it:

```go
import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-common-handler/models/identity"
)
```

**Step 2: Add FieldRagID constant**

In `bin-ai-manager/models/ai/field.go`, add after `FieldEngineKey`:

```go
FieldEngineKey   Field = "engine_key"
FieldRagID       Field = "rag_id"

FieldInitPrompt Field = "init_prompt"
```

**Step 3: Add RagID to WebhookMessage and ConvertWebhookMessage**

In `bin-ai-manager/models/ai/webhook.go`, add `RagID` to the struct after `EngineKey`:

```go
EngineKey   string         `json:"engine_key,omitempty"`
RagID       uuid.UUID      `json:"rag_id,omitempty"`

InitPrompt string `json:"init_prompt,omitempty"`
```

Add `uuid` import:
```go
import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-ai-manager/models/tool"
)
```

In `ConvertWebhookMessage()`, add `RagID: h.RagID` after `EngineKey`:

```go
EngineKey:   h.EngineKey,
RagID:       h.RagID,

InitPrompt: h.InitPrompt,
```

**Step 4: Verify compile (tests come after handler changes)**

Run:
```bash
cd bin-ai-manager && go build ./cmd/...
```
Expected: Build succeeds. Full test run deferred to after handler signature updates.

---

### Task 4: Add search_knowledge tool constants

**Files:**
- Modify: `bin-ai-manager/models/tool/main.go:7-31`
- Modify: `bin-ai-manager/models/message/tool.go:27-39`

**Step 1: Add ToolNameSearchKnowledge**

In `bin-ai-manager/models/tool/main.go`, add after `ToolNameStopService`:

```go
ToolNameStopMedia         ToolName = "stop_media"
ToolNameStopService       ToolName = "stop_service"
ToolNameSearchKnowledge   ToolName = "search_knowledge"
```

Add to `AllToolNames` slice:

```go
var AllToolNames = []ToolName{
	ToolNameConnectCall,
	ToolNameGetVariables,
	ToolNameGetAIcallMessages,
	ToolNameSendEmail,
	ToolNameSendMessage,
	ToolNameSetVariables,
	ToolNameStopFlow,
	ToolNameStopMedia,
	ToolNameStopService,
	ToolNameSearchKnowledge,
}
```

**Step 2: Add FunctionCallNameSearchKnowledge**

In `bin-ai-manager/models/message/tool.go`, add after `FunctionCallNameStopService`:

```go
FunctionCallNameStopFlow          FunctionCallName = "stop_flow"
FunctionCallNameStopService       FunctionCallName = "stop_service"
FunctionCallNameSearchKnowledge   FunctionCallName = "search_knowledge"
```

**Step 3: Verify compile**

Run: `cd bin-ai-manager && go build ./cmd/...`

---

### Task 5: Add search_knowledge tool definition

**Files:**
- Modify: `bin-ai-manager/pkg/toolhandler/definitions.go:407-444`

**Step 1: Add tool definition**

After the `get_aicall_messages` tool definition (the last entry, closing `}` on line 443), add:

```go
	{
		Name: tool.ToolNameSearchKnowledge,
		Description: `Searches the configured knowledge base for information relevant to the user's question.

WHEN TO USE:
- User asks a question that might be answered by company documentation, FAQs, or product guides
- User needs specific information about products, services, policies, or procedures
- User references something that would be in uploaded documents
- You need factual information to answer accurately rather than relying on general knowledge

WHEN NOT TO USE:
- General conversation or greetings
- Questions you can confidently answer from the conversation context
- User explicitly asks you NOT to look things up
- The question is about the current call or conversation state (use get_variables instead)

EXAMPLES:
- User: "What are your pricing plans?" -> search_knowledge(query="pricing plans and tiers")
- User: "How do I reset my password?" -> search_knowledge(query="password reset procedure")
- User: "What's your return policy?" -> search_knowledge(query="return and refund policy")
- User: "Tell me about the enterprise plan features" -> search_knowledge(query="enterprise plan features and capabilities")

run_llm: Always set true — you should respond to the user based on the search results.`,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"run_llm": map[string]any{
					"type":        "boolean",
					"description": "Always set true to respond based on search results.",
					"default":     true,
				},
				"query": map[string]any{
					"type":        "string",
					"description": "The search query to find relevant information in the knowledge base. Rephrase the user's question as a clear search query for better results.",
				},
			},
			"required": []string{"query"},
		},
	},
```

**Step 2: Verify compile**

Run: `cd bin-ai-manager && go build ./cmd/...`

---

### Task 6: Add ragID parameter to AIHandler interface and implementation

**Files:**
- Modify: `bin-ai-manager/pkg/aihandler/main.go:21-55` (interface)
- Modify: `bin-ai-manager/pkg/aihandler/chatbot.go:15-96` (implementation)
- Modify: `bin-ai-manager/pkg/aihandler/db.go:15-149` (db functions)

**Step 1: Add ragID to AIHandler interface**

In `bin-ai-manager/pkg/aihandler/main.go`, add `ragID uuid.UUID` after `engineKey string` in both `Create` and `Update`:

```go
Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineModel ai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
) (*ai.AI, error)
```

```go
Update(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	engineModel ai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoice string,
	sttType ai.STTType,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
) (*ai.AI, error)
```

**Step 2: Add ragID to chatbot.go Create/Update implementations**

In `bin-ai-manager/pkg/aihandler/chatbot.go`, add `ragID uuid.UUID` after `engineKey string` in both function signatures, and pass it to `dbCreate`/`dbUpdate`:

Create (line 15-54):
```go
func (h *aiHandler) Create(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineModel ai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
) (*ai.AI, error) {
	// ... validation unchanged ...

	res, err := h.dbCreate(ctx, customerID, name, detail, engineModel, parameter, engineKey, ragID, initPrompt, ttsType, ttsVoiceID, sttType, toolNames, vadConfig, smartTurnEnabled)
	// ...
}
```

Update (line 57-96):
```go
func (h *aiHandler) Update(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	engineModel ai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
) (*ai.AI, error) {
	// ... validation unchanged ...

	res, err := h.dbUpdate(ctx, id, name, detail, engineModel, parameter, engineKey, ragID, initPrompt, ttsType, ttsVoiceID, sttType, toolNames, vadConfig, smartTurnEnabled)
	// ...
}
```

**Step 3: Add ragID to db.go dbCreate/dbUpdate**

In `bin-ai-manager/pkg/aihandler/db.go`, add `ragID uuid.UUID` after `engineKey string` in both functions.

In `dbCreate` (line 15-69), add the parameter and set it in the struct:

```go
func (h *aiHandler) dbCreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineModel ai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoiceID string,
	sttType ai.STTType,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
) (*ai.AI, error) {
	id := h.utilHandler.UUIDCreate()
	c := &ai.AI{
		// ... Identity unchanged ...

		Name:   name,
		Detail: detail,

		EngineModel: engineModel,
		Parameter:   parameter,
		EngineKey:   engineKey,
		RagID:       ragID,

		InitPrompt: initPrompt,

		TTSType:    ttsType,
		TTSVoiceID: ttsVoiceID,

		STTType: sttType,

		ToolNames: toolNames,

		VADConfig:        vadConfig,
		SmartTurnEnabled: smartTurnEnabled,
	}
	// ... rest unchanged ...
}
```

In `dbUpdate` (line 107-149), add the parameter and field mapping:

```go
func (h *aiHandler) dbUpdate(
	ctx context.Context,
	id uuid.UUID,
	name string,
	detail string,
	engineModel ai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType ai.TTSType,
	ttsVoice string,
	sttType ai.STTType,
	toolNames []tool.ToolName,
	vadConfig *ai.VADConfig,
	smartTurnEnabled bool,
) (*ai.AI, error) {
	fields := map[ai.Field]any{
		ai.FieldName:        name,
		ai.FieldDetail:      detail,
		ai.FieldEngineModel: engineModel,
		ai.FieldParameter:   parameter,
		ai.FieldEngineKey:   engineKey,
		ai.FieldRagID:       ragID,
		ai.FieldInitPrompt:  initPrompt,
		ai.FieldTTSType:     ttsType,
		ai.FieldTTSVoiceID:  ttsVoice,
		ai.FieldSTTType:     sttType,
		ai.FieldToolNames:        toolNames,
		ai.FieldVADConfig:        vadConfig,
		ai.FieldSmartTurnEnabled: smartTurnEnabled,
	}
	// ... rest unchanged ...
}
```

**Step 4: Verify compile**

Run: `cd bin-ai-manager && go build ./cmd/...`

Expected: Will fail because listenhandler call sites haven't been updated yet. That's Task 7.

---

### Task 7: Update AI Manager listenhandler request structs and handlers

**Files:**
- Modify: `bin-ai-manager/pkg/listenhandler/models/request/ais.go:13-57`
- Modify: `bin-ai-manager/pkg/listenhandler/v1_ais.go:92-107,218-233`

**Step 1: Add RagID to request structs**

In `bin-ai-manager/pkg/listenhandler/models/request/ais.go`:

`V1DataAIsPost` — add after `EngineKey`:
```go
EngineKey   string         `json:"engine_key,omitempty"`
RagID       uuid.UUID      `json:"rag_id,omitempty"`

InitPrompt string `json:"init_prompt,omitempty"`
```

Add `uuid` import:
```go
import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
)
```

`V1DataAIsIDPut` — same addition after `EngineKey`:
```go
EngineKey   string         `json:"engine_key,omitempty"`
RagID       uuid.UUID      `json:"rag_id,omitempty"`

InitPrompt string `json:"init_prompt,omitempty"`
```

**Step 2: Pass RagID in listenhandler Create/Update calls**

In `bin-ai-manager/pkg/listenhandler/v1_ais.go`:

`processV1AIsPost` (line 92-107) — add `req.RagID` after `req.EngineKey`:
```go
tmp, err := h.aiHandler.Create(
	ctx,
	req.CustomerID,
	req.Name,
	req.Detail,
	req.EngineModel,
	req.Parameter,
	req.EngineKey,
	req.RagID,
	req.InitPrompt,
	req.TTSType,
	req.TTSVoiceID,
	req.STTType,
	req.ToolNames,
	req.VADConfig,
	req.SmartTurnEnabled,
)
```

`processV1AIsIDPut` (line 218-233) — add `req.RagID` after `req.EngineKey`:
```go
tmp, err := h.aiHandler.Update(
	ctx,
	id,
	req.Name,
	req.Detail,
	req.EngineModel,
	req.Parameter,
	req.EngineKey,
	req.RagID,
	req.InitPrompt,
	req.TTSType,
	req.TTSVoiceID,
	req.STTType,
	req.ToolNames,
	req.VADConfig,
	req.SmartTurnEnabled,
)
```

**Step 3: Verify compile**

Run: `cd bin-ai-manager && go build ./cmd/...`

Expected: Build succeeds within ai-manager. Full verification deferred to after bin-common-handler update.

---

### Task 8: Add ragID to bin-common-handler RequestHandler interface and implementation

**Files:**
- Modify: `bin-common-handler/pkg/requesthandler/main.go:188-216`
- Modify: `bin-common-handler/pkg/requesthandler/ai_ais.go:63-114,138-188`

**Step 1: Add ragID to RequestHandler interface signatures**

In `bin-common-handler/pkg/requesthandler/main.go`:

`AIV1AICreate` (line 188-201) — add `ragID uuid.UUID` after `engineKey string`:
```go
AIV1AICreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineModel amai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType amai.TTSType,
	ttsVoiceID string,
	sttType amai.STTType,
	toolNames []amtool.ToolName,
) (*amai.AI, error)
```

`AIV1AIUpdate` (line 203-216) — add `ragID uuid.UUID` after `engineKey string`:
```go
AIV1AIUpdate(
	ctx context.Context,
	aiID uuid.UUID,
	name string,
	detail string,
	engineModel amai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType amai.TTSType,
	ttsVoiceID string,
	sttType amai.STTType,
	toolNames []amtool.ToolName,
) (*amai.AI, error)
```

**Step 2: Add ragID to ai_ais.go function implementations**

In `bin-common-handler/pkg/requesthandler/ai_ais.go`:

`AIV1AICreate` (line 63-114) — add `ragID uuid.UUID` after `engineKey string`, populate request struct:
```go
func (r *requestHandler) AIV1AICreate(
	ctx context.Context,
	customerID uuid.UUID,
	name string,
	detail string,
	engineModel amai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType amai.TTSType,
	ttsVoiceID string,
	sttType amai.STTType,
	toolNames []amtool.ToolName,
) (*amai.AI, error) {
	uri := "/v1/ais"

	data := &amrequest.V1DataAIsPost{
		CustomerID: customerID,
		Name:       name,
		Detail:     detail,

		EngineModel: engineModel,
		Parameter:   parameter,
		EngineKey:   engineKey,
		RagID:       ragID,

		InitPrompt: initPrompt,

		TTSType:    ttsType,
		TTSVoiceID: ttsVoiceID,

		STTType: sttType,

		ToolNames: toolNames,
	}
	// ... rest unchanged ...
}
```

`AIV1AIUpdate` (line 138-188) — add `ragID uuid.UUID` after `engineKey string`, populate request struct:
```go
func (r *requestHandler) AIV1AIUpdate(
	ctx context.Context,
	aiID uuid.UUID,
	name string,
	detail string,
	engineModel amai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType amai.TTSType,
	ttsVoiceID string,
	sttType amai.STTType,
	toolNames []amtool.ToolName,
) (*amai.AI, error) {
	uri := fmt.Sprintf("/v1/ais/%s", aiID)

	data := &amrequest.V1DataAIsIDPut{
		Name:   name,
		Detail: detail,

		EngineModel: engineModel,
		Parameter:   parameter,
		EngineKey:   engineKey,
		RagID:       ragID,

		InitPrompt: initPrompt,

		TTSType:    ttsType,
		TTSVoiceID: ttsVoiceID,

		STTType: sttType,

		ToolNames: toolNames,
	}
	// ... rest unchanged ...
}
```

**Step 3: Regenerate mocks**

```bash
cd bin-common-handler && go generate ./pkg/requesthandler/...
```

**Step 4: Run verification for bin-common-handler**

```bash
cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass. No existing tests call `AIV1AICreate`/`AIV1AIUpdate` in bin-common-handler.

**Step 5: Commit**

```bash
git add bin-common-handler/
git commit -m "NOJIRA-Wire-rag-to-pipecat-call

- bin-common-handler: Add ragID parameter to AIV1AICreate and AIV1AIUpdate RPC functions"
```

---

### Task 9: Add toolHandleSearchKnowledge to AI Manager

**Files:**
- Modify: `bin-ai-manager/pkg/aicallhandler/tool.go:44-54` (dispatch map)
- Modify: `bin-ai-manager/pkg/aicallhandler/tool.go` (add new handler function at end of file)

**Step 1: Add dispatch map entry**

In `bin-ai-manager/pkg/aicallhandler/tool.go`, add to `mapFunctions` (after line 53):

```go
message.FunctionCallNameStopMedia:         h.toolHandleMediaStop,
message.FunctionCallNameStopService:       h.toolHandleServiceStop,
message.FunctionCallNameSearchKnowledge:   h.toolHandleSearchKnowledge,
```

**Step 2: Add import for strings package**

Add `"strings"` to the import block if not already present:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)
```

**Step 3: Add handler function**

Append to end of file (after `toolHandleGetAIcallMessages`):

```go
func (h *aicallHandler) toolHandleSearchKnowledge(ctx context.Context, c *aicall.AIcall, tc *message.ToolCall) *messageContent {
	log := logrus.WithFields(logrus.Fields{
		"func":      "toolHandleSearchKnowledge",
		"aicall_id": c.ID,
	})
	log.Debugf("handling tool search_knowledge.")

	res := newToolResult(tc.ID)

	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		fillFailed(res, err)
		return res
	}

	ai, err := h.aiHandler.Get(ctx, c.AssistanceID)
	if err != nil {
		log.Errorf("Could not get AI. err: %v", err)
		fillFailed(res, fmt.Errorf("could not retrieve AI configuration"))
		return res
	}
	log.WithField("ai", ai).Debugf("Retrieved AI info. ai_id: %s", ai.ID)

	if ai.RagID == uuid.Nil {
		fillFailed(res, fmt.Errorf("no knowledge base is configured for this assistant"))
		return res
	}

	ragRes, err := h.reqHandler.RagV1RagQuery(ctx, ai.RagID, args.Query, 5)
	if err != nil {
		log.Errorf("RAG query failed. err: %v", err)
		fillFailed(res, fmt.Errorf("knowledge base search failed"))
		return res
	}
	log.Debugf("RAG query completed. rag_id: %s, source_count: %d", ai.RagID, len(ragRes.Sources))

	if len(ragRes.Sources) == 0 {
		fillSuccess(res, "rag", ai.RagID.String(), "No relevant information found in the knowledge base.")
		return res
	}

	var sb strings.Builder
	for i, s := range ragRes.Sources {
		fmt.Fprintf(&sb, "[Source %d: \"%s\" > \"%s\" (relevance: %.2f)]\n",
			i+1, s.DocumentName, s.SectionTitle, s.RelevanceScore)
		sb.WriteString(s.Text)
		sb.WriteString("\n\n")
	}

	fillSuccess(res, "rag", ai.RagID.String(), sb.String())
	return res
}
```

**Step 4: Run verification for bin-ai-manager**

```bash
cd bin-ai-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Expected: All pass.

**Step 5: Commit**

```bash
git add bin-ai-manager/
git commit -m "NOJIRA-Wire-rag-to-pipecat-call

- bin-ai-manager: Add RagID field to AI model, field constants, and webhook
- bin-ai-manager: Add search_knowledge tool constant, function call name, and definition
- bin-ai-manager: Add ragID parameter to AIHandler Create/Update interface and implementation
- bin-ai-manager: Add ragID to listenhandler request structs and handler calls
- bin-ai-manager: Add toolHandleSearchKnowledge to tool dispatch"
```

---

### Task 10: Add ragID to bin-api-manager ServiceHandler

**Files:**
- Modify: `bin-api-manager/pkg/servicehandler/ai.go:35-91,223-288`
- Modify: `bin-api-manager/pkg/servicehandler/main.go` (interface, if AICreate/AIUpdate are defined there)

**Step 1: Check where AICreate/AIUpdate are in the interface**

Search for the interface definition containing `AICreate` in `bin-api-manager/pkg/servicehandler/main.go`. Add `ragID uuid.UUID` after `engineKey string` to both `AICreate` and `AIUpdate` signatures.

**Step 2: Add ragID to AICreate implementation**

In `bin-api-manager/pkg/servicehandler/ai.go`, `AICreate` (line 35-91):

Add `ragID uuid.UUID` after `engineKey string` in the function signature:

```go
func (h *serviceHandler) AICreate(
	ctx context.Context,
	a *amagent.Agent,
	name string,
	detail string,
	engineModel amai.EngineModel,
	parameter map[string]any,
	engineKey string,
	ragID uuid.UUID,
	initPrompt string,
	ttsType amai.TTSType,
	ttsVoiceID string,
	sttType amai.STTType,
	toolNames []amtool.ToolName,
) (*amai.WebhookMessage, error) {
```

Add RAG ownership validation before the RPC call:

```go
if !h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionCustomerAdmin|amagent.PermissionCustomerManager) {
	log.Info("The user has no permission for this agent.")
	return nil, fmt.Errorf("user has no permission")
}

// validate RAG ownership
if ragID != uuid.Nil {
	rag, err := h.reqHandler.RagV1RagGet(ctx, ragID)
	if err != nil {
		log.Errorf("Could not get RAG. err: %v", err)
		return nil, fmt.Errorf("could not validate knowledge base")
	}
	if rag.CustomerID != a.CustomerID {
		log.Infof("RAG customer_id mismatch. rag_customer_id: %s, agent_customer_id: %s", rag.CustomerID, a.CustomerID)
		return nil, fmt.Errorf("knowledge base does not belong to this customer")
	}
}

tmp, err := h.reqHandler.AIV1AICreate(
	ctx,
	a.CustomerID,
	name,
	detail,
	engineModel,
	parameter,
	engineKey,
	ragID,
	initPrompt,
	ttsType,
	ttsVoiceID,
	sttType,
	toolNames,
)
```

Add log field `"rag_id": ragID` to the log fields.

**Step 3: Add ragID to AIUpdate implementation**

In `bin-api-manager/pkg/servicehandler/ai.go`, `AIUpdate` (line 223-288):

Add `ragID uuid.UUID` after `engineKey string` in the function signature. Add the same RAG ownership validation between permission check and RPC call:

```go
// validate RAG ownership
if ragID != uuid.Nil {
	rag, err := h.reqHandler.RagV1RagGet(ctx, ragID)
	if err != nil {
		log.Errorf("Could not get RAG. err: %v", err)
		return nil, fmt.Errorf("could not validate knowledge base")
	}
	if rag.CustomerID != a.CustomerID {
		log.Infof("RAG customer_id mismatch. rag_customer_id: %s, agent_customer_id: %s", rag.CustomerID, a.CustomerID)
		return nil, fmt.Errorf("knowledge base does not belong to this customer")
	}
}

tmp, err := h.reqHandler.AIV1AIUpdate(
	ctx,
	id,
	name,
	detail,
	engineModel,
	parameter,
	engineKey,
	ragID,
	initPrompt,
	ttsType,
	ttsVoiceID,
	sttType,
	toolNames,
)
```

Add log field `"rag_id": ragID` to the log fields.

**Step 4: Check if RagV1RagGet exists in requesthandler**

Verify that `RagV1RagGet` exists in `bin-common-handler/pkg/requesthandler/`. If not, check what RAG get function is available and use that. The function name should follow the pattern `RagV1RagGet(ctx, ragID)`.

**Step 5: Update call sites in api-manager handlers that call AICreate/AIUpdate**

Search for `AICreate` and `AIUpdate` calls in `bin-api-manager` (likely in a handler file that parses HTTP request bodies). Add `ragID` (parsed from request body) to those calls.

**Step 6: Run verification**

```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 7: Commit**

```bash
git add bin-api-manager/
git commit -m "NOJIRA-Wire-rag-to-pipecat-call

- bin-api-manager: Add ragID parameter to AICreate and AIUpdate with cross-tenant RAG ownership validation"
```

---

### Task 11: Filter search_knowledge tool in pipecat-manager when RagID is nil

**Files:**
- Modify: `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:76-82`

**Step 1: Add tool filtering after tool resolution**

In `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go`, after the line `tools = h.toolHandler.GetByNames(ai.ToolNames)` (line 81), add:

```go
tools = h.toolHandler.GetByNames(ai.ToolNames)

// filter out search_knowledge if no RAG is configured
if ai.RagID == uuid.Nil {
	filtered := make([]aitool.Tool, 0, len(tools))
	for _, t := range tools {
		if t.Name != aitool.ToolNameSearchKnowledge {
			filtered = append(filtered, t)
		}
	}
	tools = filtered
}
```

Check that `uuid` is imported. Looking at the existing imports — `"github.com/gofrs/uuid"` is already imported (line 18).

**Step 2: Run verification**

```bash
cd bin-pipecat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 3: Commit**

```bash
git add bin-pipecat-manager/
git commit -m "NOJIRA-Wire-rag-to-pipecat-call

- bin-pipecat-manager: Filter out search_knowledge tool when AI has no RAG configured"
```

---

### Task 12: Update OpenAPI schema

**Files:**
- Modify: `bin-openapi-manager/openapi/openapi.yaml` (AIManagerAI schema)
- Modify: `bin-openapi-manager/openapi/paths/ais/main.yaml` (create request)
- Modify: `bin-openapi-manager/openapi/paths/ais/id.yaml` (update request)

**Step 1: Add rag_id to AIManagerAI schema**

In `bin-openapi-manager/openapi/openapi.yaml`, locate the `AIManagerAI` schema and add `rag_id` after `engine_key`:

```yaml
rag_id:
  type: string
  format: uuid
  x-go-type: string
  description: "The knowledge base ID for the search_knowledge tool. Obtained from the `id` field of `GET /rags`."
  example: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
```

**Step 2: Add rag_id to create and update request schemas**

Add `rag_id` to the request body schemas in `main.yaml` (POST) and `id.yaml` (PUT) — NOT required, optional field.

**Step 3: Read bin-openapi-manager/CLAUDE.md BEFORE making changes**

IMPORTANT: Read `bin-openapi-manager/CLAUDE.md` for AI-Native Specification Rules before modifying the OpenAPI files.

**Step 4: Regenerate and verify**

```bash
cd bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

Then regenerate api-manager:
```bash
cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 5: Commit**

```bash
git add bin-openapi-manager/ bin-api-manager/
git commit -m "NOJIRA-Wire-rag-to-pipecat-call

- bin-openapi-manager: Add rag_id field to AIManagerAI schema and create/update request bodies
- bin-api-manager: Regenerate server code from updated OpenAPI spec"
```

---

### Task 13: Update RST documentation

**Files:**
- Modify: `bin-api-manager/docsdev/source/ai_struct_ai.rst`

**Step 1: Add rag_id field documentation**

Add `rag_id` to the AI struct documentation after `engine_key`:

```rst
* ``rag_id`` (UUID, Optional): The knowledge base ID for the ``search_knowledge`` tool.
  Obtained from the ``id`` field of ``GET https://api.voipbin.net/v1.0/rags``.
  When set, the AI assistant can search this knowledge base during voice calls.
  Set to ``00000000-0000-0000-0000-000000000000`` or omit to disable.
```

**Step 2: Rebuild HTML**

```bash
cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build
```

**Step 3: Stage and commit**

```bash
git add bin-api-manager/docsdev/source/ai_struct_ai.rst
git add -f bin-api-manager/docsdev/build/
git commit -m "NOJIRA-Wire-rag-to-pipecat-call

- bin-api-manager: Add rag_id field to AI struct RST documentation"
```

---

### Task 14: Final verification across all changed services

**Step 1: Verify each changed service**

Run in sequence:

```bash
cd bin-rag-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ../bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ../bin-ai-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ../bin-pipecat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ../bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

cd ../bin-openapi-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

**Step 2: Check for other services calling AIV1AICreate/AIV1AIUpdate**

Since the bin-common-handler `RequestHandler` interface signature changed, any service that uses `AIV1AICreate` or `AIV1AIUpdate` will fail to compile. Search the monorepo:

```bash
grep -r "AIV1AICreate\|AIV1AIUpdate" --include="*.go" -l | grep -v vendor | grep -v mock | grep -v _test.go
```

Only `bin-api-manager` and `bin-common-handler` should appear. If any other services call these functions, update them too.

**Step 3: Final commit if needed**

If any fixes were needed, commit them.

---

### Task 15: Squash and push

**Step 1: Review all commits**

```bash
git log --oneline main..HEAD
```

**Step 2: Push branch**

```bash
git push -u origin NOJIRA-Wire-rag-to-pipecat-call
```

**Step 3: Create PR**

Title: `NOJIRA-Wire-rag-to-pipecat-call`

Body:
```
Wire RAG (knowledge base) to pipecat voice calls via a search_knowledge tool.
AI assistants can now search customer knowledge bases during live calls.

- bin-rag-manager: Add Text field to query.Source so RAG responses include chunk text
- bin-dbscheme-manager: Add rag_id column to ai_ais table
- bin-common-handler: Add ragID parameter to AIV1AICreate and AIV1AIUpdate RPC functions
- bin-ai-manager: Add RagID to AI model, search_knowledge tool definition and handler
- bin-api-manager: Add ragID to ServiceHandler with cross-tenant RAG ownership validation
- bin-openapi-manager: Add rag_id to AIManagerAI OpenAPI schema
- bin-pipecat-manager: Filter search_knowledge tool when AI has no RAG configured
```

---

## Deployment Order

1. **bin-rag-manager** — backward-compatible additive field
2. **Database migration** — `rag_id` column, nullable, existing rows unaffected
3. **bin-common-handler** — RPC signature change (only bin-api-manager calls these)
4. **bin-ai-manager** — model + tools + handler
5. **bin-openapi-manager** — schema update
6. **bin-api-manager** — ServiceHandler + validation + generated code
7. **bin-pipecat-manager** — tool filtering

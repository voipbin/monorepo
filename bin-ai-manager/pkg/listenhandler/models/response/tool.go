package response

import (
	"monorepo/bin-ai-manager/models/tool"
)

// V1ToolsGet is the response for GET /v1/tools
type V1ToolsGet struct {
	Tools []tool.Tool `json:"tools"`
}

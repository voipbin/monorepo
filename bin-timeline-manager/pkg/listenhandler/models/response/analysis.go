package response

import (
	"monorepo/bin-timeline-manager/models/analysis"
)

// V1DataAnalysesList represents the list response.
type V1DataAnalysesList struct {
	Result        []*analysis.Analysis `json:"result"`
	NextPageToken string               `json:"next_page_token,omitempty"`
}

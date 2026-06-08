package response

import (
	"monorepo/bin-timeline-manager/models/correlation"
)

// V1DataResourceCorrelationGet represents the correlation graph for a resource.
// It is a type alias of the single source-of-truth transport contract in
// models/correlation to avoid struct drift between the listenhandler response
// and the requesthandler client. The listenhandler remains the layer that
// constructs this DTO from the domain correlation.ResourceCorrelation.
type V1DataResourceCorrelationGet = correlation.ResourceCorrelationResponse

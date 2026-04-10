package analyzer

import (
	"fmt"
	"sort"
)

// LayerRule defines which layers a given layer is allowed to depend on.
// Dependencies not in the allowed set are violations.
var LayerRule = map[Layer][]Layer{
	// Core can depend on other Core and Integration
	LayerCore: {LayerCore, LayerIntegration, LayerProxy, LayerTelephony},

	// Telephony can depend on Core, Integration, Proxy
	LayerTelephony: {LayerCore, LayerTelephony, LayerIntegration, LayerProxy},

	// Business can depend on Core, Telephony, Integration, Messaging
	LayerBusiness: {LayerCore, LayerBusiness, LayerTelephony, LayerIntegration, LayerMessaging},

	// Messaging can depend on Core, Integration
	LayerMessaging: {LayerCore, LayerMessaging, LayerIntegration},

	// Integration can depend on Core
	LayerIntegration: {LayerCore, LayerIntegration},

	// Gateway can depend on Core, Business, Telephony, Messaging, Integration
	LayerGateway: {LayerCore, LayerGateway, LayerBusiness, LayerTelephony, LayerMessaging, LayerIntegration},

	// Proxy should only depend on Core and other Proxies
	LayerProxy: {LayerCore, LayerProxy},

	// Tooling can depend on anything (utility layer)
	LayerTooling: {LayerCore, LayerTelephony, LayerBusiness, LayerMessaging, LayerIntegration, LayerGateway, LayerProxy, LayerTooling},
}

// LayerViolation represents a dependency that crosses layer boundaries.
type LayerViolation struct {
	From      string
	FromLayer Layer
	To        string
	ToLayer   Layer
	DepType   DependencyType
}

// String formats the violation for display.
func (v LayerViolation) String() string {
	return fmt.Sprintf("[%s] %s (%s) -> %s (%s)",
		v.DepType, v.From, v.FromLayer, v.To, v.ToLayer)
}

// DetectLayerViolations checks all dependencies against the layer rules
// and returns any that violate the expected hierarchy.
func DetectLayerViolations(g *Graph, layerMap map[string]Layer) []LayerViolation {
	var violations []LayerViolation

	for _, dep := range g.Dependencies {
		fromLayer, fromOK := layerMap[dep.From]
		toLayer, toOK := layerMap[dep.To]

		if !fromOK || !toOK {
			continue // skip unknown services
		}

		if !isLayerAllowed(fromLayer, toLayer) {
			violations = append(violations, LayerViolation{
				From:      dep.From,
				FromLayer: fromLayer,
				To:        dep.To,
				ToLayer:   toLayer,
				DepType:   dep.Type,
			})
		}
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].FromLayer != violations[j].FromLayer {
			return violations[i].FromLayer < violations[j].FromLayer
		}
		return violations[i].From < violations[j].From
	})

	return violations
}

func isLayerAllowed(from, to Layer) bool {
	allowed, exists := LayerRule[from]
	if !exists {
		return true // unknown layer, allow by default
	}
	for _, l := range allowed {
		if l == to {
			return true
		}
	}
	return false
}

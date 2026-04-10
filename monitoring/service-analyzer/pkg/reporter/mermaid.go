package reporter

import (
	"fmt"
	"sort"
	"strings"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
)

// layerAssignment maps service names to architectural layers.
var layerAssignment = map[string]analyzer.Layer{
	"common-handler": analyzer.LayerCore,
	"call-manager":   analyzer.LayerCore,
	"flow-manager":   analyzer.LayerCore,
	"customer-manager": analyzer.LayerCore,

	"conference-manager":  analyzer.LayerTelephony,
	"transfer-manager":    analyzer.LayerTelephony,
	"registrar-manager":   analyzer.LayerTelephony,
	"tts-manager":         analyzer.LayerTelephony,
	"transcribe-manager":  analyzer.LayerTelephony,
	"pipecat-manager":     analyzer.LayerTelephony,

	"agent-manager":   analyzer.LayerBusiness,
	"billing-manager": analyzer.LayerBusiness,
	"campaign-manager": analyzer.LayerBusiness,
	"queue-manager":    analyzer.LayerBusiness,
	"outdial-manager":  analyzer.LayerBusiness,

	"message-manager":      analyzer.LayerMessaging,
	"email-manager":        analyzer.LayerMessaging,
	"talk-manager":         analyzer.LayerMessaging,
	"conversation-manager": analyzer.LayerMessaging,

	"hook-manager":    analyzer.LayerIntegration,
	"webhook-manager": analyzer.LayerIntegration,
	"storage-manager": analyzer.LayerIntegration,
	"number-manager":  analyzer.LayerIntegration,

	"api-manager":      analyzer.LayerGateway,
	"ai-manager":       analyzer.LayerGateway,
	"tag-manager":      analyzer.LayerGateway,
	"sentinel-manager": analyzer.LayerGateway,
	"direct-manager":   analyzer.LayerGateway,
	"timeline-manager": analyzer.LayerGateway,
	"route-manager":    analyzer.LayerGateway,
	"contact-manager":  analyzer.LayerGateway,
	"rag-manager":      analyzer.LayerGateway,

	"asterisk-proxy":  analyzer.LayerProxy,
	"rtpengine-proxy": analyzer.LayerProxy,

	"openapi-manager":  analyzer.LayerTooling,
	"dbscheme-manager": analyzer.LayerTooling,
}

// GenerateMermaid produces a Mermaid graph definition from the dependency graph.
func GenerateMermaid(g *analyzer.Graph) string {
	var sb strings.Builder
	sb.WriteString("graph TD\n")

	// group services by layer
	layers := make(map[analyzer.Layer][]string)
	for _, svc := range g.Services {
		layer, ok := layerAssignment[svc.Name]
		if !ok {
			layer = analyzer.LayerGateway
		}
		layers[layer] = append(layers[layer], svc.Name)
	}

	layerOrder := []analyzer.Layer{
		analyzer.LayerCore,
		analyzer.LayerTelephony,
		analyzer.LayerBusiness,
		analyzer.LayerMessaging,
		analyzer.LayerIntegration,
		analyzer.LayerGateway,
		analyzer.LayerProxy,
		analyzer.LayerTooling,
	}

	for _, layer := range layerOrder {
		svcs, ok := layers[layer]
		if !ok || len(svcs) == 0 {
			continue
		}
		sort.Strings(svcs)
		sb.WriteString(fmt.Sprintf("    subgraph %s[\"%s Layer\"]\n", sanitizeID(string(layer)), layer))
		for _, svc := range svcs {
			sb.WriteString(fmt.Sprintf("        %s[%s]\n", sanitizeID(svc), svc))
		}
		sb.WriteString("    end\n\n")
	}

	// edges
	for _, dep := range g.Dependencies {
		fromID := sanitizeID(dep.From)
		toID := sanitizeID(dep.To)
		if dep.Type == analyzer.DepRPC {
			sb.WriteString(fmt.Sprintf("    %s -->|RPC| %s\n", fromID, toID))
		} else {
			sb.WriteString(fmt.Sprintf("    %s -.->|event| %s\n", fromID, toID))
		}
	}

	return sb.String()
}

func sanitizeID(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

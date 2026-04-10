package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// methodPrefixToService maps requesthandler method prefixes to target service names.
var methodPrefixToService = map[string]string{
	"AI":           "ai-manager",
	"Agent":        "agent-manager",
	"Billing":      "billing-manager",
	"Call":         "call-manager",
	"Campaign":     "campaign-manager",
	"Conference":   "conference-manager",
	"Contact":      "contact-manager",
	"Conversation": "conversation-manager",
	"Customer":     "customer-manager",
	"Direct":       "direct-manager",
	"Email":        "email-manager",
	"Flow":         "flow-manager",
	"Hook":         "hook-manager",
	"Message":      "message-manager",
	"Number":       "number-manager",
	"Outdial":      "outdial-manager",
	"Pipecat":      "pipecat-manager",
	"Queue":        "queue-manager",
	"Rag":          "rag-manager",
	"RTPEngine":    "rtpengine-proxy",
	"Registrar":    "registrar-manager",
	"Route":        "route-manager",
	"Storage":      "storage-manager",
	"Tag":          "tag-manager",
	"Talk":         "talk-manager",
	"Timeline":     "timeline-manager",
	"Transcribe":   "transcribe-manager",
	"Transfer":     "transfer-manager",
	"TTS":          "tts-manager",
	"Webhook":      "webhook-manager",
}

// rpcCallPattern matches reqHandler.SomeV1Method( calls in Go source.
var rpcCallPattern = regexp.MustCompile(`\.(?:reqHandler|requestHandler|h\.reqHandler|h\.requestHandler)\.([A-Z]\w+V1\w+)\(`)

// publishEventPattern matches notifyHandler.PublishEvent(ctx, xxx.EventTypeYYY, data) calls.
var publishEventPattern = regexp.MustCompile(`\.PublishEvent\w*\(.*?,\s*(\w+\.EventType\w+)`)

// subscriberPublisherPattern matches m.Publisher == string(commonoutline.ServiceNameXxx) in switch/case.
var subscriberPublisherPattern = regexp.MustCompile(`ServiceName(\w+)`)

// Scanner analyzes Go source files to extract service dependencies.
type Scanner struct {
	monorepoRoot string
}

// NewScanner creates a scanner rooted at the given monorepo path.
func NewScanner(monorepoRoot string) *Scanner {
	return &Scanner{monorepoRoot: monorepoRoot}
}

// DiscoverServices finds all bin-* and voip-* service directories.
func (s *Scanner) DiscoverServices() ([]Service, error) {
	var services []Service

	entries, err := os.ReadDir(s.monorepoRoot)
	if err != nil {
		return nil, fmt.Errorf("read monorepo root: %w", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "bin-") || strings.HasPrefix(name, "voip-") {
			services = append(services, Service{
				Name:      extractServiceName(name),
				Directory: filepath.Join(s.monorepoRoot, name),
			})
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})
	return services, nil
}

// ScanRPCDependencies finds all RPC calls each service makes to other services.
func (s *Scanner) ScanRPCDependencies(services []Service) ([]Dependency, error) {
	depMap := make(map[string]map[string][]string) // from -> to -> methods

	for _, svc := range services {
		pkgDir := filepath.Join(svc.Directory, "pkg")
		if _, err := os.Stat(pkgDir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			// skip test and mock files
			if strings.HasSuffix(path, "_test.go") || strings.Contains(path, "mock_") {
				return nil
			}

			methods, scanErr := scanFileForRPCCalls(path)
			if scanErr != nil {
				return nil
			}

			fromService := svc.Name
			for _, method := range methods {
				target := resolveMethodTarget(method)
				if target == "" || target == fromService {
					continue
				}
				if depMap[fromService] == nil {
					depMap[fromService] = make(map[string][]string)
				}
				depMap[fromService][target] = appendUnique(depMap[fromService][target], method)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", svc.Directory, err)
		}
	}

	var deps []Dependency
	for from, targets := range depMap {
		for to, methods := range targets {
			sort.Strings(methods)
			deps = append(deps, Dependency{
				From:    from,
				To:      to,
				Type:    DepRPC,
				Methods: methods,
			})
		}
	}

	sort.Slice(deps, func(i, j int) bool {
		if deps[i].From != deps[j].From {
			return deps[i].From < deps[j].From
		}
		return deps[i].To < deps[j].To
	})
	return deps, nil
}

// serviceNameToService maps ServiceName constant suffixes to service names.
var serviceNameToService = map[string]string{
	"AIManager":           "ai-manager",
	"AgentManager":        "agent-manager",
	"AsteriskProxy":       "asterisk-proxy",
	"BillingManager":      "billing-manager",
	"CallManager":         "call-manager",
	"CampaignManager":     "campaign-manager",
	"ConferenceManager":   "conference-manager",
	"ContactManager":      "contact-manager",
	"ConversationManager": "conversation-manager",
	"CustomerManager":     "customer-manager",
	"DirectManager":       "direct-manager",
	"EmailManager":        "email-manager",
	"FlowManager":         "flow-manager",
	"HookManager":         "hook-manager",
	"MessageManager":      "message-manager",
	"NumberManager":       "number-manager",
	"OutdialManager":      "outdial-manager",
	"PipecatManager":      "pipecat-manager",
	"QueueManager":        "queue-manager",
	"RagManager":          "rag-manager",
	"RegistrarManager":    "registrar-manager",
	"RouteManager":        "route-manager",
	"SentinelManager":     "sentinel-manager",
	"StorageManager":      "storage-manager",
	"TagManager":          "tag-manager",
	"TalkManager":         "talk-manager",
	"TimelineManager":     "timeline-manager",
	"TranscribeManager":   "transcribe-manager",
	"TransferManager":     "transfer-manager",
	"TTSManager":          "tts-manager",
	"WebhookManager":      "webhook-manager",
}

// ScanEventDependencies finds event publish/subscribe relationships.
// It scans subscribehandler directories for ServiceName references to identify
// which publisher each service subscribes to.
func (s *Scanner) ScanEventDependencies(services []Service) ([]Dependency, error) {
	depMap := make(map[string]map[string][]string) // subscriber -> publisher -> event types

	for _, svc := range services {
		subDir := filepath.Join(svc.Directory, "pkg", "subscribehandler")
		if _, err := os.Stat(subDir); os.IsNotExist(err) {
			continue
		}

		// find which publishers this service subscribes to
		publisherRefs, err := scanDirForPublisherRefs(subDir)
		if err != nil {
			continue
		}

		subscriberName := svc.Name
		for _, pubRef := range publisherRefs {
			publisherName, ok := serviceNameToService[pubRef]
			if !ok || publisherName == subscriberName {
				continue
			}
			if depMap[subscriberName] == nil {
				depMap[subscriberName] = make(map[string][]string)
			}
			depMap[subscriberName][publisherName] = appendUnique(
				depMap[subscriberName][publisherName], "event_subscription")
		}

		// also scan for specific EventType references
		eventTypes, err := scanDirForEventTypes(subDir)
		if err == nil {
			for publisher, events := range eventTypes {
				if publisher == subscriberName {
					continue
				}
				if depMap[subscriberName] == nil {
					depMap[subscriberName] = make(map[string][]string)
				}
				for _, evt := range events {
					depMap[subscriberName][publisher] = appendUnique(
						depMap[subscriberName][publisher], evt)
				}
			}
		}
	}

	var deps []Dependency
	for subscriber, publishers := range depMap {
		for publisher, events := range publishers {
			sort.Strings(events)
			deps = append(deps, Dependency{
				From:    subscriber,
				To:      publisher,
				Type:    DepEvent,
				Methods: events,
			})
		}
	}

	sort.Slice(deps, func(i, j int) bool {
		if deps[i].From != deps[j].From {
			return deps[i].From < deps[j].From
		}
		return deps[i].To < deps[j].To
	})
	return deps, nil
}

// BuildGraph constructs the complete dependency graph.
func (s *Scanner) BuildGraph() (*Graph, error) {
	services, err := s.DiscoverServices()
	if err != nil {
		return nil, err
	}

	rpcDeps, err := s.ScanRPCDependencies(services)
	if err != nil {
		return nil, fmt.Errorf("scan rpc: %w", err)
	}

	eventDeps, err := s.ScanEventDependencies(services)
	if err != nil {
		return nil, fmt.Errorf("scan events: %w", err)
	}

	var allDeps []Dependency
	allDeps = append(allDeps, rpcDeps...)
	allDeps = append(allDeps, eventDeps...)

	return &Graph{
		Services:     services,
		Dependencies: allDeps,
	}, nil
}

// ComputeMetrics calculates fan-in/fan-out for each service.
func ComputeMetrics(g *Graph) []ServiceMetrics {
	metricsMap := make(map[string]*ServiceMetrics)
	for _, svc := range g.Services {
		metricsMap[svc.Name] = &ServiceMetrics{Name: svc.Name}
	}

	for _, dep := range g.Dependencies {
		if dep.Type == DepRPC {
			if m, ok := metricsMap[dep.From]; ok {
				m.RPCFanOut++
				m.RPCTargets = appendUnique(m.RPCTargets, dep.To)
			}
			if m, ok := metricsMap[dep.To]; ok {
				m.RPCFanIn++
				m.RPCCallers = appendUnique(m.RPCCallers, dep.From)
			}
		} else if dep.Type == DepEvent {
			if m, ok := metricsMap[dep.From]; ok {
				m.EventConsumers += len(dep.Methods)
			}
			if m, ok := metricsMap[dep.To]; ok {
				m.EventPublishers += len(dep.Methods)
			}
		}
	}

	var result []ServiceMetrics
	for _, m := range metricsMap {
		sort.Strings(m.RPCTargets)
		sort.Strings(m.RPCCallers)
		result = append(result, *m)
	}

	sort.Slice(result, func(i, j int) bool {
		totalI := result[i].RPCFanIn + result[i].RPCFanOut
		totalJ := result[j].RPCFanIn + result[j].RPCFanOut
		return totalI > totalJ
	})
	return result
}

// --- helpers ---

func extractServiceName(dirName string) string {
	for _, prefix := range []string{"bin-", "voip-"} {
		if strings.HasPrefix(dirName, prefix) {
			return strings.TrimPrefix(dirName, prefix)
		}
	}
	return dirName
}

func scanFileForRPCCalls(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var methods []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		matches := rpcCallPattern.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				methods = append(methods, m[1])
			}
		}
	}
	return methods, scanner.Err()
}

func resolveMethodTarget(method string) string {
	for prefix, service := range methodPrefixToService {
		if strings.HasPrefix(method, prefix+"V1") {
			return service
		}
	}
	return ""
}

// scanDirForPublisherRefs finds ServiceName<Xxx> references in subscribehandler files.
func scanDirForPublisherRefs(dir string) ([]string, error) {
	var refs []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || strings.Contains(filepath.Base(path), "mock_") {
			return nil
		}

		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			matches := subscriberPublisherPattern.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) > 1 {
					refs = appendUnique(refs, m[1])
				}
			}
		}
		return nil
	})
	return refs, err
}

// eventTypeImportPattern matches event type constant references like cucustomer.EventTypeCustomerDeleted
var eventTypeImportPattern = regexp.MustCompile(`(\w+)\.EventType(\w+)`)

// scanDirForEventTypes finds EventType references and maps them to publisher services
// based on import alias conventions (e.g., cucustomer → customer-manager).
func scanDirForEventTypes(dir string) (map[string][]string, error) {
	result := make(map[string][]string)
	// import alias prefix → service name mapping
	aliasToService := map[string]string{
		"cucustomer":    "customer-manager",
		"cscustomer":    "customer-manager",
		"fmactiveflow":  "flow-manager",
		"fmflow":        "flow-manager",
		"cmcall":        "call-manager",
		"cmconfbridge":  "call-manager",
		"cmrecording":   "call-manager",
		"cmchannel":     "call-manager",
		"cmgroupcall":   "call-manager",
		"dtmf":          "call-manager",
		"confbridge":    "call-manager",
		"ari":           "call-manager",
		"mmmessage":     "message-manager",
		"ememail":       "email-manager",
		"nmnumber":      "number-manager",
		"smpod":         "sentinel-manager",
		"tmspeaking":    "tts-manager",
		"tmstreaming":   "tts-manager",
		"amaicall":      "ai-manager",
		"ammessage":     "ai-manager",
		"qmqueuecall":   "queue-manager",
		"pmpipecatcall": "pipecat-manager",
		"amagent":       "agent-manager",
		"wmwebhook":     "webhook-manager",
		"tmtranscribe":  "transcribe-manager",
		"tmtransfer":    "transfer-manager",
		"tkchat":        "talk-manager",
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") || strings.Contains(filepath.Base(path), "mock_") {
			return nil
		}

		f, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer f.Close()

		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			matches := eventTypeImportPattern.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) > 2 {
					alias := m[1]
					eventName := m[2]
					if svc, ok := aliasToService[alias]; ok {
						result[svc] = appendUnique(result[svc], eventName)
					}
				}
			}
		}
		return nil
	})
	return result, err
}

func appendUnique(slice []string, val string) []string {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}

package analyzer

// Service represents a microservice in the monorepo.
type Service struct {
	Name      string
	Directory string
}

// DependencyType classifies how one service depends on another.
type DependencyType string

const (
	DepRPC   DependencyType = "rpc"
	DepEvent DependencyType = "event"
)

// Dependency represents a directed edge: From calls/subscribes To.
type Dependency struct {
	From     string
	To       string
	Type     DependencyType
	Methods  []string // RPC method names or event types
}

// ServiceMetrics holds fan-in/fan-out counts for a single service.
type ServiceMetrics struct {
	Name            string
	RPCFanOut       int      // services this one calls via RPC
	RPCFanIn        int      // services that call this one via RPC
	EventPublishers int      // event types this service publishes
	EventConsumers  int      // event types this service subscribes to
	RPCTargets      []string // names of RPC targets
	RPCCallers      []string // names of services calling this one
}

// Graph holds the full dependency graph for the monorepo.
type Graph struct {
	Services     []Service
	Dependencies []Dependency
}

// Layer classifies a service by its position in the architecture.
type Layer string

const (
	LayerCore        Layer = "Core"
	LayerTelephony   Layer = "Telephony"
	LayerBusiness    Layer = "Business"
	LayerMessaging   Layer = "Messaging"
	LayerIntegration Layer = "Integration"
	LayerGateway     Layer = "Gateway"
	LayerProxy       Layer = "Proxy"
	LayerTooling     Layer = "Tooling"
)

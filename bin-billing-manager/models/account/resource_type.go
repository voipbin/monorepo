package account

// ResourceType define
type ResourceType string

// list of resource types
const (
	ResourceTypeExtension     ResourceType = "extension"
	ResourceTypeAgent         ResourceType = "agent"
	ResourceTypeQueue         ResourceType = "queue"
	ResourceTypeFlow          ResourceType = "flow"
	ResourceTypeConference    ResourceType = "conference"
	ResourceTypeTrunk         ResourceType = "trunk"
	ResourceTypeVirtualNumber ResourceType = "virtual_number"
)

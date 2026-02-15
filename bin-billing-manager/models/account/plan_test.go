package account

import (
	"testing"
)

func TestPlanLimits_GetLimit(t *testing.T) {
	tests := []struct {
		name         string
		limits       PlanLimits
		resourceType ResourceType
		expected     int
	}{
		{
			name: "get extensions limit",
			limits: PlanLimits{
				Extensions:     10,
				Agents:         5,
				Queues:         3,
				Flows:          0,
				Conferences:    2,
				Trunks:         1,
				VirtualNumbers: 8,
			},
			resourceType: ResourceTypeExtension,
			expected:     10,
		},
		{
			name: "get agents limit",
			limits: PlanLimits{
				Extensions:     10,
				Agents:         5,
				Queues:         3,
				Flows:          0,
				Conferences:    2,
				Trunks:         1,
				VirtualNumbers: 8,
			},
			resourceType: ResourceTypeAgent,
			expected:     5,
		},
		{
			name: "get queues limit",
			limits: PlanLimits{
				Extensions:     10,
				Agents:         5,
				Queues:         3,
				Flows:          0,
				Conferences:    2,
				Trunks:         1,
				VirtualNumbers: 8,
			},
			resourceType: ResourceTypeQueue,
			expected:     3,
		},
		{
			name: "get flows limit",
			limits: PlanLimits{
				Extensions:     10,
				Agents:         5,
				Queues:         3,
				Flows:          0,
				Conferences:    2,
				Trunks:         1,
				VirtualNumbers: 8,
			},
			resourceType: ResourceTypeFlow,
			expected:     0,
		},
		{
			name: "get conferences limit",
			limits: PlanLimits{
				Extensions:     10,
				Agents:         5,
				Queues:         3,
				Flows:          0,
				Conferences:    2,
				Trunks:         1,
				VirtualNumbers: 8,
			},
			resourceType: ResourceTypeConference,
			expected:     2,
		},
		{
			name: "get trunks limit",
			limits: PlanLimits{
				Extensions:     10,
				Agents:         5,
				Queues:         3,
				Flows:          0,
				Conferences:    2,
				Trunks:         1,
				VirtualNumbers: 8,
			},
			resourceType: ResourceTypeTrunk,
			expected:     1,
		},
		{
			name: "get virtual numbers limit",
			limits: PlanLimits{
				Extensions:     10,
				Agents:         5,
				Queues:         3,
				Flows:          0,
				Conferences:    2,
				Trunks:         1,
				VirtualNumbers: 8,
			},
			resourceType: ResourceTypeVirtualNumber,
			expected:     8,
		},
		{
			name: "unknown resource type",
			limits: PlanLimits{
				Extensions:     10,
				Agents:         5,
				Queues:         3,
				Flows:          0,
				Conferences:    2,
				Trunks:         1,
				VirtualNumbers: 8,
			},
			resourceType: ResourceType("unknown"),
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.limits.GetLimit(tt.resourceType)
			if result != tt.expected {
				t.Errorf("GetLimit() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestPlanTokenMap(t *testing.T) {
	tests := []struct {
		name     string
		planType PlanType
		expected int64
	}{
		{"free plan", PlanTypeFree, 1000},
		{"basic plan", PlanTypeBasic, 10000},
		{"professional plan", PlanTypeProfessional, 100000},
		{"unlimited plan", PlanTypeUnlimited, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, exists := PlanTokenMap[tt.planType]
			if !exists {
				t.Errorf("PlanTokenMap missing entry for %s", tt.planType)
			}
			if result != tt.expected {
				t.Errorf("PlanTokenMap[%s] = %d, expected %d", tt.planType, result, tt.expected)
			}
		})
	}
}

func TestPlanLimitMap(t *testing.T) {
	tests := []struct {
		name     string
		planType PlanType
	}{
		{"free plan exists", PlanTypeFree},
		{"basic plan exists", PlanTypeBasic},
		{"professional plan exists", PlanTypeProfessional},
		{"unlimited plan exists", PlanTypeUnlimited},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, exists := PlanLimitMap[tt.planType]
			if !exists {
				t.Errorf("PlanLimitMap missing entry for %s", tt.planType)
			}
		})
	}
}

func TestPlanLimitMapValues(t *testing.T) {
	tests := []struct {
		name           string
		planType       PlanType
		expectedLimits PlanLimits
	}{
		{
			name:     "free plan limits",
			planType: PlanTypeFree,
			expectedLimits: PlanLimits{
				Extensions:     5,
				Agents:         5,
				Queues:         2,
				Flows:          0,
				Conferences:    2,
				Trunks:         1,
				VirtualNumbers: 5,
			},
		},
		{
			name:     "basic plan limits",
			planType: PlanTypeBasic,
			expectedLimits: PlanLimits{
				Extensions:     50,
				Agents:         50,
				Queues:         10,
				Flows:          0,
				Conferences:    10,
				Trunks:         5,
				VirtualNumbers: 50,
			},
		},
		{
			name:     "professional plan limits",
			planType: PlanTypeProfessional,
			expectedLimits: PlanLimits{
				Extensions:     500,
				Agents:         500,
				Queues:         100,
				Flows:          0,
				Conferences:    100,
				Trunks:         50,
				VirtualNumbers: 500,
			},
		},
		{
			name:     "unlimited plan limits",
			planType: PlanTypeUnlimited,
			expectedLimits: PlanLimits{
				Extensions:     0,
				Agents:         0,
				Queues:         0,
				Flows:          0,
				Conferences:    0,
				Trunks:         0,
				VirtualNumbers: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := PlanLimitMap[tt.planType]
			if limits.Extensions != tt.expectedLimits.Extensions {
				t.Errorf("Extensions = %d, expected %d", limits.Extensions, tt.expectedLimits.Extensions)
			}
			if limits.Agents != tt.expectedLimits.Agents {
				t.Errorf("Agents = %d, expected %d", limits.Agents, tt.expectedLimits.Agents)
			}
			if limits.Queues != tt.expectedLimits.Queues {
				t.Errorf("Queues = %d, expected %d", limits.Queues, tt.expectedLimits.Queues)
			}
			if limits.Flows != tt.expectedLimits.Flows {
				t.Errorf("Flows = %d, expected %d", limits.Flows, tt.expectedLimits.Flows)
			}
			if limits.Conferences != tt.expectedLimits.Conferences {
				t.Errorf("Conferences = %d, expected %d", limits.Conferences, tt.expectedLimits.Conferences)
			}
			if limits.Trunks != tt.expectedLimits.Trunks {
				t.Errorf("Trunks = %d, expected %d", limits.Trunks, tt.expectedLimits.Trunks)
			}
			if limits.VirtualNumbers != tt.expectedLimits.VirtualNumbers {
				t.Errorf("VirtualNumbers = %d, expected %d", limits.VirtualNumbers, tt.expectedLimits.VirtualNumbers)
			}
		})
	}
}

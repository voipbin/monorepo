package teamhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"monorepo/bin-ai-manager/models/team"
)

func Test_validateTeam(t *testing.T) {
	memberA := uuid.FromStringOrNil("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	memberB := uuid.FromStringOrNil("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	aiA := uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111")
	aiB := uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222")

	validMembers := []team.Member{
		{
			ID:   memberA,
			Name: "Greeter",
			AIID: aiA,
			Transitions: []team.Transition{
				{FunctionName: "transfer_to_b", Description: "Go to B", NextMemberID: memberB},
			},
		},
		{
			ID:   memberB,
			Name: "Specialist",
			AIID: aiB,
			Transitions: []team.Transition{
				{FunctionName: "transfer_to_a", Description: "Go to A", NextMemberID: memberA},
			},
		},
	}

	tests := []struct {
		name          string
		startMemberID uuid.UUID
		members       []team.Member
		wantErr       bool
		errContains   string
	}{
		{
			name:          "valid team",
			startMemberID: memberA,
			members:       validMembers,
		},
		{
			name:          "empty members",
			startMemberID: memberA,
			members:       []team.Member{},
			wantErr:       true,
			errContains:   "members list must not be empty",
		},
		{
			name:          "nil start member id",
			startMemberID: uuid.Nil,
			members:       validMembers,
			wantErr:       true,
			errContains:   "start_member_id must not be empty",
		},
		{
			name:          "start member id not in members",
			startMemberID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999"),
			members:       validMembers,
			wantErr:       true,
			errContains:   "not found in members",
		},
		{
			name:          "duplicate member ids",
			startMemberID: memberA,
			members: []team.Member{
				{ID: memberA, Name: "A", AIID: aiA},
				{ID: memberA, Name: "B", AIID: aiB},
			},
			wantErr:     true,
			errContains: "duplicate member id",
		},
		{
			name:          "nil member id",
			startMemberID: memberA,
			members: []team.Member{
				{ID: uuid.Nil, Name: "A", AIID: aiA},
			},
			wantErr:     true,
			errContains: "member id must not be empty",
		},
		{
			name:          "nil member ai_id",
			startMemberID: memberA,
			members: []team.Member{
				{ID: memberA, Name: "A", AIID: uuid.Nil},
			},
			wantErr:     true,
			errContains: "member ai_id must not be empty",
		},
		{
			name:          "empty member name",
			startMemberID: memberA,
			members: []team.Member{
				{ID: memberA, Name: "", AIID: aiA},
			},
			wantErr:     true,
			errContains: "member name must not be empty",
		},
		{
			name:          "transition function name collides with reserved tool",
			startMemberID: memberA,
			members: []team.Member{
				{
					ID: memberA, Name: "A", AIID: aiA,
					Transitions: []team.Transition{
						{FunctionName: "connect_call", Description: "bad", NextMemberID: memberA},
					},
				},
			},
			wantErr:     true,
			errContains: "collides with reserved tool name",
		},
		{
			name:          "duplicate function name within member",
			startMemberID: memberA,
			members: []team.Member{
				{
					ID: memberA, Name: "A", AIID: aiA,
					Transitions: []team.Transition{
						{FunctionName: "do_thing", Description: "first", NextMemberID: memberA},
						{FunctionName: "do_thing", Description: "second", NextMemberID: memberA},
					},
				},
			},
			wantErr:     true,
			errContains: "duplicate transition function_name",
		},
		{
			name:          "transition next member id not found",
			startMemberID: memberA,
			members: []team.Member{
				{
					ID: memberA, Name: "A", AIID: aiA,
					Transitions: []team.Transition{
						{FunctionName: "go_nowhere", Description: "bad", NextMemberID: uuid.FromStringOrNil("99999999-9999-9999-9999-999999999999")},
					},
				},
			},
			wantErr:     true,
			errContains: "next_member_id",
		},
		{
			name:          "same function name on different members is allowed",
			startMemberID: memberA,
			members: []team.Member{
				{
					ID: memberA, Name: "A", AIID: aiA,
					Transitions: []team.Transition{
						{FunctionName: "transfer_back", Description: "go to B", NextMemberID: memberB},
					},
				},
				{
					ID: memberB, Name: "B", AIID: aiB,
					Transitions: []team.Transition{
						{FunctionName: "transfer_back", Description: "go to A", NextMemberID: memberA},
					},
				},
			},
		},
		{
			name:          "single member no transitions",
			startMemberID: memberA,
			members: []team.Member{
				{ID: memberA, Name: "Solo", AIID: aiA},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTeam(tt.startMemberID, tt.members)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

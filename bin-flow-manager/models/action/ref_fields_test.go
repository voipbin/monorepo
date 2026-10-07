package action

import (
	"reflect"
	"sort"
	"testing"
)

func Test_RefFieldsOf(t *testing.T) {
	tests := []struct {
		name string
		ty   Type
		want []RefField
	}{
		{
			name: "goto has one ref:action scalar field",
			ty:   TypeGoto,
			want: []RefField{{JSONName: "target_id", Kind: RefKindAction}},
		},
		{
			name: "branch has a scalar and a map ref:action field",
			ty:   TypeBranch,
			want: []RefField{
				{JSONName: "default_target_id", Kind: RefKindAction},
				{JSONName: "target_ids", Kind: RefKindAction, IsMap: true},
			},
		},
		{
			name: "condition_datetime has one ref:action false_target_id",
			ty:   TypeConditionDatetime,
			want: []RefField{{JSONName: "false_target_id", Kind: RefKindAction}},
		},
		{
			name: "queue_join has one ref:resource field",
			ty:   TypeQueueJoin,
			want: []RefField{{JSONName: "queue_id", Kind: RefKindResource}},
		},
		{
			name: "ai_summary has two ref:resource fields",
			ty:   TypeAISummary,
			want: []RefField{
				{JSONName: "on_end_flow_id", Kind: RefKindResource},
				{JSONName: "reference_id", Kind: RefKindResource},
			},
		},
		{
			name: "hangup has one ref:resource field",
			ty:   TypeHangup,
			want: []RefField{{JSONName: "reference_id", Kind: RefKindResource}},
		},
		{
			name: "connect has an address and an address list",
			ty:   TypeConnect,
			want: []RefField{
				{JSONName: "source", Kind: RefKindAddress},
				{JSONName: "destinations", Kind: RefKindAddress, IsList: true},
			},
		},
		{
			name: "answer has no option fields at all",
			ty:   TypeAnswer,
			want: nil,
		},
		{
			name: "mute maps to struct{}{}, not a struct with fields",
			ty:   TypeMute,
			want: nil,
		},
		{
			name: "unknown type returns nil",
			ty:   Type("does_not_exist"),
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RefFieldsOf(tt.ty)
			sortRefFields(got)
			sortRefFields(tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RefFieldsOf(%q) = %+v, want %+v", tt.ty, got, tt.want)
			}
		})
	}
}

func sortRefFields(fs []RefField) {
	sort.Slice(fs, func(i, j int) bool { return fs[i].JSONName < fs[j].JSONName })
}

package variable

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestVariableStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	variables := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	v := Variable{
		ID:        id,
		Variables: variables,
	}

	if v.ID != id {
		t.Errorf("Variable.ID = %v, expected %v", v.ID, id)
	}
	if len(v.Variables) != 2 {
		t.Errorf("Variable.Variables length = %v, expected %v", len(v.Variables), 2)
	}
	if v.Variables["key1"] != "value1" {
		t.Errorf("Variable.Variables[key1] = %v, expected %v", v.Variables["key1"], "value1")
	}
	if v.Variables["key2"] != "value2" {
		t.Errorf("Variable.Variables[key2] = %v, expected %v", v.Variables["key2"], "value2")
	}
}

func TestVariableWithNilVariables(t *testing.T) {
	v := Variable{
		ID: uuid.Must(uuid.NewV4()),
	}

	if v.Variables != nil {
		t.Errorf("Variable.Variables should be nil, got %v", v.Variables)
	}
}

func TestVariableWithEmptyVariables(t *testing.T) {
	v := Variable{
		ID:        uuid.Must(uuid.NewV4()),
		Variables: map[string]string{},
	}

	if len(v.Variables) != 0 {
		t.Errorf("Variable.Variables length = %v, expected %v", len(v.Variables), 0)
	}
}

func TestVariableModification(t *testing.T) {
	v := Variable{
		ID:        uuid.Must(uuid.NewV4()),
		Variables: map[string]string{"existing": "value"},
	}

	v.Variables["new_key"] = "new_value"

	if len(v.Variables) != 2 {
		t.Errorf("Variable.Variables length = %v, expected %v after adding", len(v.Variables), 2)
	}
	if v.Variables["new_key"] != "new_value" {
		t.Errorf("Variable.Variables[new_key] = %v, expected %v", v.Variables["new_key"], "new_value")
	}
}

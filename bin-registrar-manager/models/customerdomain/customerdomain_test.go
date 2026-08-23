package customerdomain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_CustomerDomainStruct(t *testing.T) {
	curTime := func() *time.Time { t := time.Date(2026, 8, 23, 1, 2, 3, 0, time.UTC); return &t }()

	cd := CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("8f2c1d16-7f7d-11ee-99f7-0b0f16f0d0a1"),
		DomainLabel: "ab12",
		Realm:       "ab12.reg.voipbin.net",
		TMCreate:    curTime,
		TMUpdate:    nil,
	}

	if cd.CustomerID.String() != "8f2c1d16-7f7d-11ee-99f7-0b0f16f0d0a1" {
		t.Errorf("Wrong match. expect: 8f2c1d16-7f7d-11ee-99f7-0b0f16f0d0a1, got: %s", cd.CustomerID)
	}
	if cd.DomainLabel != "ab12" {
		t.Errorf("Wrong match. expect: ab12, got: %s", cd.DomainLabel)
	}
	if cd.Realm != "ab12.reg.voipbin.net" {
		t.Errorf("Wrong match. expect: ab12.reg.voipbin.net, got: %s", cd.Realm)
	}
	if cd.TMCreate != curTime {
		t.Errorf("Wrong match. expect: %v, got: %v", curTime, cd.TMCreate)
	}
	if cd.TMUpdate != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", cd.TMUpdate)
	}
}

func Test_CustomerDomainJSONTags(t *testing.T) {
	cd := CustomerDomain{
		CustomerID:  uuid.FromStringOrNil("9a41f3c2-7f7d-11ee-8f5a-df1f6f8dbb31"),
		DomainLabel: "x9z0",
		Realm:       "x9z0.reg.voipbin.net",
	}

	data, err := json.Marshal(cd)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	tmp := map[string]any{}
	if errUnmarshal := json.Unmarshal(data, &tmp); errUnmarshal != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", errUnmarshal)
	}

	expectKeys := []string{"customer_id", "domain_label", "realm", "tm_create", "tm_update"}
	for _, key := range expectKeys {
		if _, ok := tmp[key]; !ok {
			t.Errorf("Wrong match. expect key: %s, got: missing", key)
		}
	}

	if tmp["domain_label"] != "x9z0" {
		t.Errorf("Wrong match. expect: x9z0, got: %v", tmp["domain_label"])
	}
	if tmp["realm"] != "x9z0.reg.voipbin.net" {
		t.Errorf("Wrong match. expect: x9z0.reg.voipbin.net, got: %v", tmp["realm"])
	}
}

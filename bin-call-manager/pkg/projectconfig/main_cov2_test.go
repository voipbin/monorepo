package projectconfig

import (
	"reflect"
	"testing"
)

func Test_Get_cov2(t *testing.T) {
	first := Get()
	if first == nil {
		t.Fatalf("Wrong match. expect: not nil, got: nil")
	}

	// singleton: same instance on every call.
	if second := Get(); second != first {
		t.Errorf("Wrong match. expect same instance, got: %p vs %p", first, second)
	}

	// the loaded values are derived from the base domain.
	expect := &ProjectConfig{
		ProjectBaseDomain:     first.ProjectBaseDomain,
		DomainConference:      "conference." + first.ProjectBaseDomain,
		DomainPSTN:            "pstn." + first.ProjectBaseDomain,
		DomainSIP:             "sip." + first.ProjectBaseDomain,
		DomainTrunkSuffix:     ".trunk." + first.ProjectBaseDomain,
		DomainRegistrarSuffix: ".reg." + first.ProjectBaseDomain,
		ProjectBucketName:     first.ProjectBucketName,
	}
	if !reflect.DeepEqual(first, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, first)
	}
}

func Test_load_cov2_env(t *testing.T) {
	t.Setenv("PROJECT_BASE_DOMAIN", "example.test")
	t.Setenv("PROJECT_BUCKET_NAME", "test-bucket")

	res := load()
	expect := &ProjectConfig{
		ProjectBaseDomain:     "example.test",
		DomainConference:      "conference.example.test",
		DomainPSTN:            "pstn.example.test",
		DomainSIP:             "sip.example.test",
		DomainTrunkSuffix:     ".trunk.example.test",
		DomainRegistrarSuffix: ".reg.example.test",
		ProjectBucketName:     "test-bucket",
	}
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("Wrong match.\nexpect: %v\ngot: %v", expect, res)
	}
}

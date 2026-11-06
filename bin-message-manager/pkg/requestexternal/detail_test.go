package requestexternal

import (
	"strings"
	"testing"
)

func Test_providerErrorDetail(t *testing.T) {
	tests := []struct {
		name string
		body string

		expectRes string
	}{
		{"telnyx", `{"errors":[{"code":"40301","title":"Not allowed","detail":"The sender is not allowed"}]}`, "code 40301 Not allowed"},
		{"telnyx detail only", `{"errors":[{"detail":"fake not found"}]}`, "fake not found"},
		{"messagebird", `{"errors":[{"code":2,"description":"request not allowed (incorrect access_key)","parameter":"access_key"}]}`, "code 2 request not allowed (incorrect access_key)"},
		{"digit runs are masked", `{"errors":[{"code":9,"description":"recipient 15555550100 is invalid, code 12345"}]}`, "code 9 recipient <num> is invalid, code 12345"},
		{"not json", `<html>+15555550100</html>`, ""},
		{"no errors", `{"ok":true}`, ""},
		{"empty", ``, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := providerErrorDetail(strings.NewReader(tt.body))
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_providerErrorDetail_truncated(t *testing.T) {
	body := `{"errors":[{"code":1,"description":"` + strings.Repeat("가", 500) + `"}]}`

	res := providerErrorDetail(strings.NewReader(body))
	if len([]rune(res)) != providerErrorDetailLimit {
		t.Errorf("Wrong length. expect: %d, got: %d", providerErrorDetailLimit, len([]rune(res)))
	}
}

func Test_ProviderError_Error(t *testing.T) {
	e := &ProviderError{Provider: "p", StatusCode: 422}
	if e.Error() != "p refused the request. status_code: 422" {
		t.Errorf("Wrong error. got: %s", e.Error())
	}

	e.Detail = "code 2 bad"
	if e.Error() != "p refused the request. status_code: 422, detail: code 2 bad" {
		t.Errorf("Wrong error. got: %s", e.Error())
	}
}

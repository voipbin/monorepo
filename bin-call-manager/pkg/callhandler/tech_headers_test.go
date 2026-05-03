package callhandler

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func Test_mergeTechHeaders(t *testing.T) {

	tests := []struct {
		name string

		dst map[string]string
		src map[string]string

		expectDst     map[string]string
		expectApplied int
		expectSkipped int
	}{
		{
			"nil src leaves dst unchanged",

			map[string]string{"CALLERID(num)": "+820000000000"},
			nil,

			map[string]string{"CALLERID(num)": "+820000000000"},
			0,
			0,
		},
		{
			"empty src leaves dst unchanged",

			map[string]string{"CALLERID(num)": "+820000000000"},
			map[string]string{},

			map[string]string{"CALLERID(num)": "+820000000000"},
			0,
			0,
		},
		{
			"normal header gets wrapped and applied",

			map[string]string{},
			map[string]string{"X-Carrier-Auth": "token-abc"},

			map[string]string{
				"PJSIP_HEADER(add,X-Carrier-Auth)": "token-abc",
			},
			1,
			0,
		},
		{
			"empty key is skipped",

			map[string]string{},
			map[string]string{"": "whatever"},

			map[string]string{},
			0,
			1,
		},
		{
			"key with CRLF is skipped",

			map[string]string{},
			map[string]string{"X-Bad\r\nInject": "v"},

			map[string]string{},
			0,
			1,
		},
		{
			"key with parenthesis is skipped (blocks PJSIP_HEADER pre-wrap)",

			map[string]string{},
			map[string]string{"PJSIP_HEADER(add,X)": "v"},

			map[string]string{},
			0,
			1,
		},
		{
			"key with comma is skipped",

			map[string]string{},
			map[string]string{"X,Evil": "v"},

			map[string]string{},
			0,
			1,
		},
		{
			"reserved wrapped key P-Asserted-Identity is skipped",

			map[string]string{},
			map[string]string{"P-Asserted-Identity": "<tel:+100>"},

			map[string]string{},
			0,
			1,
		},
		{
			"reserved wrapped key Privacy is skipped",

			map[string]string{},
			map[string]string{"Privacy": "id"},

			map[string]string{},
			0,
			1,
		},
		{
			"reserved wrapped key SDP-Transport is skipped",

			map[string]string{},
			map[string]string{"VBOUT-SDP_Transport": "RTP/AVP"},

			map[string]string{},
			0,
			1,
		},
		{
			"reserved raw key CALLERID(name) is skipped",

			map[string]string{},
			map[string]string{"CALLERID(name)": "foo"},

			map[string]string{},
			0,
			1,
		},
		{
			"value with CR is skipped",

			map[string]string{},
			map[string]string{"X-Header": "ok\rbad"},

			map[string]string{},
			0,
			1,
		},
		{
			"value with LF is skipped",

			map[string]string{},
			map[string]string{"X-Header": "ok\nbad"},

			map[string]string{},
			0,
			1,
		},
		{
			"mixed valid and invalid — valid applied, invalid counted skipped",

			map[string]string{},
			map[string]string{
				"X-Good":              "ok",
				"":                    "drop-empty-key",
				"X-Bad\r\nInject":     "drop-bad-key",
				"P-Asserted-Identity": "drop-reserved",
			},

			map[string]string{
				"PJSIP_HEADER(add,X-Good)": "ok",
			},
			1,
			3,
		},
		{
			"reserved VBOUT-CODECS header is blocked",

			map[string]string{},
			map[string]string{"VBOUT-CODECS": "PCMU"},

			map[string]string{},
			0,
			1,
		},
		{
			"pre-existing dst key is overwritten by tech header (merge semantics; system fns re-overwrite later in createChannelOutgoing)",

			map[string]string{
				"PJSIP_HEADER(add,X-Carrier-Auth)": "old",
			},
			map[string]string{
				"X-Carrier-Auth": "new",
			},

			map[string]string{
				"PJSIP_HEADER(add,X-Carrier-Auth)": "new",
			},
			1,
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logrus.WithField("test", tt.name)

			applied, skipped := mergeTechHeaders(tt.dst, tt.src, log)

			if applied != tt.expectApplied {
				t.Errorf("Wrong applied count. expect: %d, got: %d", tt.expectApplied, applied)
			}
			if skipped != tt.expectSkipped {
				t.Errorf("Wrong skipped count. expect: %d, got: %d", tt.expectSkipped, skipped)
			}

			if len(tt.dst) != len(tt.expectDst) {
				t.Errorf("Wrong dst size. expect: %d, got: %d. dst=%v", len(tt.expectDst), len(tt.dst), tt.dst)
			}
			for k, v := range tt.expectDst {
				if got, ok := tt.dst[k]; !ok || got != v {
					t.Errorf("Wrong dst entry. key=%s expect=%q got=%q (present=%v)", k, v, got, ok)
				}
			}
		})
	}
}

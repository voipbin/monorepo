package telnyx

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_MessageEvent_GetMedias(t *testing.T) {
	tests := []struct {
		name string
		body string

		expectMedias  []PayloadMedia
		expectSkipped int
		expectMMS     bool
	}{
		{
			name:      "sms with an empty list",
			body:      `{"data":{"payload":{"type":"SMS","media":[]}}}`,
			expectMMS: false,
		},
		{
			name:      "sms without the field",
			body:      `{"data":{"payload":{"type":"SMS"}}}`,
			expectMMS: false,
		},
		{
			name:      "null list",
			body:      `{"data":{"payload":{"type":"SMS","media":null}}}`,
			expectMMS: false,
		},
		{
			name: "mms with two media",
			body: `{"data":{"payload":{"type":"MMS","media":[
				{"url":"https://media.telnyx.com/example/a.jpg","content_type":"image/jpeg","size":123,"sha256":"aa"},
				{"url":"https://media.telnyx.com/example/b.png","content_type":"image/png","size":456,"sha256":"bb","unknown":"x"}]}}}`,
			expectMedias: []PayloadMedia{
				{URL: "https://media.telnyx.com/example/a.jpg", ContentType: "image/jpeg", Size: 123, SHA256: "aa"},
				{URL: "https://media.telnyx.com/example/b.png", ContentType: "image/png", Size: 456, SHA256: "bb"},
			},
			expectMMS: true,
		},
		{
			name:      "type mms is case insensitive and has no media",
			body:      `{"data":{"payload":{"type":"mms","media":[]}}}`,
			expectMMS: true,
		},
		{
			name: "type sms but media present is an mms",
			body: `{"data":{"payload":{"type":"SMS","media":[{"url":"https://media.telnyx.com/example/a.jpg","content_type":"image/jpeg","size":1}]}}}`,
			expectMedias: []PayloadMedia{
				{URL: "https://media.telnyx.com/example/a.jpg", ContentType: "image/jpeg", Size: 1},
			},
			expectMMS: true,
		},
		{
			name: "media without size",
			body: `{"data":{"payload":{"type":"MMS","media":[{"url":"https://media.telnyx.com/example/a.jpg","content_type":"image/jpeg"}]}}}`,
			expectMedias: []PayloadMedia{
				{URL: "https://media.telnyx.com/example/a.jpg", ContentType: "image/jpeg"},
			},
			expectMMS: true,
		},
		{
			name:          "media is an object",
			body:          `{"data":{"payload":{"type":"MMS","media":{"url":"x"}}}}`,
			expectSkipped: 1,
			expectMMS:     true,
		},
		{
			name: "size is a string and content type is null in other items",
			body: `{"data":{"payload":{"type":"MMS","media":[
				{"url":"https://media.telnyx.com/example/a.jpg","content_type":"image/jpeg","size":"big"},
				{"url":"https://media.telnyx.com/example/b.png","content_type":null,"size":5}]}}}`,
			expectMedias: []PayloadMedia{
				{URL: "https://media.telnyx.com/example/b.png", Size: 5},
			},
			expectSkipped: 1,
			expectMMS:     true,
		},
		{
			name:          "item is not an object",
			body:          `{"data":{"payload":{"type":"SMS","media":["x", 5]}}}`,
			expectSkipped: 2,
			expectMMS:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ev MessageEvent
			if err := json.Unmarshal([]byte(tt.body), &ev); err != nil {
				t.Fatalf("Wrong match. the event must always parse. got: %v", err)
			}

			medias, skipped := ev.GetMedias()
			if !reflect.DeepEqual(medias, tt.expectMedias) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectMedias, medias)
			}
			if skipped != tt.expectSkipped {
				t.Errorf("Wrong match. expect skipped: %d, got: %d", tt.expectSkipped, skipped)
			}
			if got := ev.IsMMS(); got != tt.expectMMS {
				t.Errorf("Wrong match. expect IsMMS: %v, got: %v", tt.expectMMS, got)
			}
		})
	}
}

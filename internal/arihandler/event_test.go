package arihandler

import (
	"testing"
)

func TestProcessEvent(t *testing.T) {
	event := "{\"type\":\"ChannelCreated\",\"timestamp\":\"2020-04-09T21:02:25.142+0000\",\"channel\":{\"id\":\"1586466145.3679\",\"name\":\"PJSIP/in-voipbin-00000730\",\"state\":\"Ring\",\"caller\":{\"name\":\"\",\"number\":\"1000003\"},\"connected\":{\"name\":\"\",\"number\":\"\"},\"accountcode\":\"\",\"dialplan\":{\"context\":\"in-voipbin\",\"exten\":\"9011441332323027\",\"priority\":1,\"app_name\":\"\",\"app_data\":\"\"},\"creationtime\":\"2020-04-09T21:02:25.142+0000\",\"language\":\"en\"},\"asterisk_id\":\"42:01:0a:a4:00:03\",\"application\":\"voipbin\"}"

	err := processEvent(event)
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

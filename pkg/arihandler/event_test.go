package arihandler

// func TestProcessEvent(t *testing.T) {
// 	event := "{\"type\":\"ChannelCreated\",\"timestamp\":\"2020-04-09T21:02:25.142+0000\",\"channel\":{\"id\":\"1586466145.3679\",\"name\":\"PJSIP/in-voipbin-00000730\",\"state\":\"Ring\",\"caller\":{\"name\":\"\",\"number\":\"1000003\"},\"connected\":{\"name\":\"\",\"number\":\"\"},\"accountcode\":\"\",\"dialplan\":{\"context\":\"in-voipbin\",\"exten\":\"9011441332323027\",\"priority\":1,\"app_name\":\"\",\"app_data\":\"\"},\"creationtime\":\"2020-04-09T21:02:25.142+0000\",\"language\":\"en\"},\"asterisk_id\":\"42:01:0a:a4:00:03\",\"application\":\"voipbin\"}"

// 	err := processEvent([]byte(event))
// 	if err != nil {
// 		t.Errorf("Wrong match. expect: ok, got: %v", err)
// 	}
// }

// func TestHandleStasisStartIncoming(t *testing.T) {
// 	m := `{"type":"StasisStart","timestamp":"2020-04-12T22:34:41.144+0000","args":["CONTEXT=in-voipbin","DOMAIN=34.90.68.237"],"channel":{"id":"1586730880.1791","name":"PJSIP/in-voipbin-00000381","state":"Up","caller":{"name":"","number":"test123"},"connected":{"name":"","number":""},"accountcode":"","dialplan":{"context":"in-voipbin","exten":"0046605844066","priority":4,"app_name":"Stasis","app_data":"voipbin,CONTEXT=in-voipbin,DOMAIN=34.90.68.237"},"creationtime":"2020-04-12T22:34:40.641+0000","language":"en"},"asterisk_id":"42:01:0a:a4:00:05","application":"voipbin"}`

// 	var evt = ari.StasisStart{}
// 	json.Unmarshal([]byte(m), &evt)
// 	if evt.Args["CONTEXT"] != "in-voipbin" {
// 		t.Errorf("Wrong match. %v", evt)
// 	}

// 	t.Logf("message: %s", evt.Args)

// 	handleStasisStartIncoming(&evt)

// }

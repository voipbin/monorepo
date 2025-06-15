package callhandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/jart/gosip/sip"
	gomock "go.uber.org/mock/gomock"
)

func Test_getRecoveryDetails(t *testing.T) {

	tests := []struct {
		name string

		messages []string

		expectedRes *recoveryDetail
	}{
		{
			name: "incoming call - asterisk is UAC",

			messages: []string{
				"INVITE sip:+821021656521@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net SIP/2.0\r\nRecord-Route: <sip:10.164.0.20;transport=tcp;r2=on;lr>\r\nRecord-Route: <sip:34.90.68.237;r2=on;lr>\r\nVia: SIP/2.0/TCP 10.164.0.20;branch=z9hG4bKbd1e.ed19480b1337ba4e2d30b5ba43e67e0f.0\r\nVia: SIP/2.0/UDP 192.168.55.139:50431;received=211.200.20.28;branch=z9hG4bK.SVr9KCtCs;rport=50431\r\nFrom: <sip:3000@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net>;tag=a3~DOCq2q\r\nTo: sip:+821021656521@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net\r\nCSeq: 22 INVITE\r\nCall-ID: n5F9oxcuN-\r\nMax-Forwards: 70\r\nSupported: replaces, outbound, gruu, path, record-aware\r\nAllow: INVITE, ACK, CANCEL, OPTIONS, BYE, REFER, NOTIFY, MESSAGE, SUBSCRIBE, INFO, PRACK, UPDATE\r\nContent-Type: application/sdp\r\nContent-Length: 329\r\nContact: <sip:3000@211.200.20.28:50431;transport=udp>;+org.linphone.specs=\"lime\"\r\nUser-Agent: LinphoneAndroid/5.2.5 ([LG-F750K (567)]0) LinphoneSDK/5.3.47 (tags/5.3.47^0)\r\nProxy-Authorization:  Digest realm=\"5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net\", nonce=\"aEmvRGhJrhjOHPHPrIBBvfNCMT32tSnP\", username=\"3000\",  uri=\"sip:+821021656521@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net\", response=\"246981c6d6662f3a4527b858fa51a3fc\", cnonce=\"fG5MyIj5rnrxwPXH\", nc=00000001, qop=auth\r\nVB-Source: 211.200.20.28\r\nVB-Transport: udp\r\nVB-Domain: 5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net\r\n\r\nv=0\r\no=3000 2213 1609 IN IP4 10.164.0.12\r\ns=Talk\r\nc=IN IP4 10.164.0.12\r\nt=0 0\r\na=rtcp-xr:rcvr-rtt=all:10000 stat-summary=loss,dup,jitt,TTL voip-metrics\r\na=record:off\r\nm=audio 28936 RTP/AVPF 0 100\r\na=rtcp-fb:* trr-int 1000\r\na=rtcp-fb:* ccm tmmbr\r\na=rtpmap:0 PCMU/8000\r\na=rtpmap:100 telephone-event/8000\r\na=sendrecv\r\na=rtcp:28937\r\n",
				"SIP/2.0 100 Trying\r\nVia: SIP/2.0/TCP 10.164.0.20;received=10.164.0.20;branch=z9hG4bKbd1e.ed19480b1337ba4e2d30b5ba43e67e0f.0\r\nVia: SIP/2.0/UDP 192.168.55.139:50431;rport=50431;received=211.200.20.28;branch=z9hG4bK.SVr9KCtCs\r\nRecord-Route: <sip:10.164.0.20;transport=tcp;lr;r2=on>\r\nRecord-Route: <sip:34.90.68.237;lr;r2=on>\r\nCall-ID: n5F9oxcuN-\r\nFrom: <sip:3000@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net>;tag=a3~DOCq2q\r\nTo: <sip:+821021656521@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net>\r\nCSeq: 22 INVITE\r\nServer: voipbin\r\nContent-Length:  0\r\n\r\n",
				"SIP/2.0 180 Ringing\r\nVia: SIP/2.0/TCP 10.164.0.20;received=10.164.0.20;branch=z9hG4bKbd1e.ed19480b1337ba4e2d30b5ba43e67e0f.0\r\nVia: SIP/2.0/UDP 192.168.55.139:50431;rport=50431;received=211.200.20.28;branch=z9hG4bK.SVr9KCtCs\r\nRecord-Route: <sip:10.164.0.20;transport=tcp;lr;r2=on>\r\nRecord-Route: <sip:34.90.68.237;lr;r2=on>\r\nCall-ID: n5F9oxcuN-\r\nFrom: <sip:3000@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net>;tag=a3~DOCq2q\r\nTo: <sip:+821021656521@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net>;tag=db803eb7-30db-4d27-973d-692195bffcae\r\nCSeq: 22 INVITE\r\nServer: voipbin\r\nContact: <sip:10.96.5.34:5060;transport=TCP>\r\nAllow: OPTIONS, REGISTER, SUBSCRIBE, NOTIFY, PUBLISH, INVITE, ACK, BYE, CANCEL, UPDATE, PRACK, INFO, MESSAGE, REFER\r\nContent-Length:  0\r\n\r\n",
				"ACK sip:10.96.5.34:5060;transport=TCP SIP/2.0\r\nVia: SIP/2.0/TCP 10.164.0.20;branch=z9hG4bKbd1e.9eaa1f3746d0aecc4d9a90b27fa6b08a.0\r\nVia: SIP/2.0/UDP 192.168.55.139:50431;received=211.200.20.28;branch=z9hG4bK.gRBrIdW9F;rport=50431\r\nFrom: <sip:3000@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net>;tag=a3~DOCq2q\r\nTo: <sip:+821021656521@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net>;tag=db803eb7-30db-4d27-973d-692195bffcae\r\nCSeq: 22 ACK\r\nCall-ID: n5F9oxcuN-\r\nMax-Forwards: 70\r\nProxy-Authorization:  Digest realm=\"5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net\", nonce=\"aEmvRGhJrhjOHPHPrIBBvfNCMT32tSnP\", username=\"3000\",  uri=\"sip:+821021656521@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net\", response=\"246981c6d6662f3a4527b858fa51a3fc\", cnonce=\"fG5MyIj5rnrxwPXH\", nc=00000001, qop=auth\r\nUser-Agent: LinphoneAndroid/5.2.5 ([LG-F750K (567)]0) LinphoneSDK/5.3.47 (tags/5.3.47^0)\r\nContent-Length: 0\r\n\r\n",
				"SIP/2.0 200 OK\r\nVia: SIP/2.0/TCP 10.164.0.20;received=10.164.0.20;branch=z9hG4bKbd1e.ed19480b1337ba4e2d30b5ba43e67e0f.0\r\nVia: SIP/2.0/UDP 192.168.55.139:50431;rport=50431;received=211.200.20.28;branch=z9hG4bK.SVr9KCtCs\r\nRecord-Route: <sip:10.164.0.20;transport=tcp;lr;r2=on>\r\nRecord-Route: <sip:34.90.68.237;lr;r2=on>\r\nCall-ID: n5F9oxcuN-\r\nFrom: <sip:3000@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net>;tag=a3~DOCq2q\r\nTo: <sip:+821021656521@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net>;tag=db803eb7-30db-4d27-973d-692195bffcae\r\nCSeq: 22 INVITE\r\nServer: voipbin\r\nAllow: OPTIONS, REGISTER, SUBSCRIBE, NOTIFY, PUBLISH, INVITE, ACK, BYE, CANCEL, UPDATE, PRACK, INFO, MESSAGE, REFER\r\nContact: <sip:10.96.5.34:5060;transport=TCP>\r\nSupported: 100rel, timer, replaces, norefersub\r\nContent-Type: application/sdp\r\nContent-Length:   529\r\n\r\nv=0\r\no=- 2213 1611 IN IP4 10.96.5.34\r\ns=asterisk-call-6dcd66c5c8-ptrw8\r\nc=IN IP4 10.96.5.34\r\nt=0 0\r\na=msid-semantic:WMS *\r\nm=audio 10166 RTP/AVPF 0 100\r\na=connection:new\r\na=setup:actpass\r\na=fingerprint:SHA-256 D8:25:1F:B5:D2:91:A2:35:AF:26:E3:0D:F3:D9:85:A4:A9:8A:78:7D:DC:02:5C:6F:16:BD:10:EB:69:FA:7F:79\r\na=rtpmap:0 PCMU/8000\r\na=rtpmap:100 telephone-event/8000\r\na=fmtp:100 0-16\r\na=ptime:20\r\na=maxptime:140\r\na=sendrecv\r\na=msid:d36a9219-e0d6-4651-aa32-115f99274f7d 997ebabd-68c3-4c51-87c5-e76bcd526e02\r\na=rtcp-fb:* transport-cc\r\n",
			},

			expectedRes: &recoveryDetail{
				RequestURI: "sip:3000@211.200.20.28:50431;transport=udp",
				Routes:     "<sip:34.90.68.237;r2=on;lr>",
				CallID:     "n5F9oxcuN-",

				FromDisplay: "",
				FromURI:     "sip:3000@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
				FromTag:     "a3~DOCq2q",

				ToDisplay: "",
				ToURI:     "sip:+821021656521@5e4a0680-804e-11ec-8477-2fea5968d85b.registrar.voipbin.net",
				ToTag:     "db803eb7-30db-4d27-973d-692195bffcae",

				CSeq: 23,
			},
		},
		{
			name: "outgoing call - asterisk is UAS",

			messages: []string{
				"INVITE sip:+821021656521@sip.telnyx.com;transport=udp SIP/2.0\r\nVia: SIP/2.0/TCP 10.96.0.15:5060;branch=z9hG4bKPj5fe38d71-f8b8-4e93-85aa-2e7af7472e1a;alias\r\nFrom: \"Anonymous\" <sip:anonymous@anonymous.invalid>;tag=2f41957b-9c9d-45d1-a18c-310ce92516ba\r\nTo: <sip:+821021656521@sip.telnyx.com>\r\nContact: <sip:anonymous@10.96.0.15:5060;transport=TCP>\r\nCall-ID: 1ced6b72-70c6-4c45-82e1-078568bf9d45\r\nCSeq: 2594 INVITE\r\nRoute: <sip:10.164.0.20:5060;lr>\r\nAllow: OPTIONS, REGISTER, SUBSCRIBE, NOTIFY, PUBLISH, INVITE, ACK, BYE, CANCEL, UPDATE, PRACK, INFO, MESSAGE, REFER\r\nSupported: 100rel, timer, replaces, norefersub, histinfo\r\nSession-Expires: 1800\r\nMin-SE: 90\r\nP-Asserted-Identity: \"Anonymous\" <sip:+821100000001@pstn.voipbin.net>\r\nPrivacy: id\r\nVBOUT-SDP_Transport: RTP/AVP\r\nMax-Forwards: 70\r\nUser-Agent: voipbin\r\nContent-Type: application/sdp\r\nContent-Length:   268\r\n\r\nv=0\r\no=- 2035369987 2035369987 IN IP4 10.96.0.15\r\ns=asterisk-call-6dcd66c5c8-6gkbt\r\nc=IN IP4 10.96.0.15\r\nt=0 0\r\nm=audio 10384 RTP/AVPF 0 101\r\na=rtpmap:0 PCMU/8000\r\na=rtpmap:101 telephone-event/8000\r\na=fmtp:101 0-16\r\na=ptime:20\r\na=maxptime:140\r\na=sendrecv\r\na=rtcp-mux\r\n",
				"SIP/2.0 100 Telnyx Trying\r\nVia: SIP/2.0/TCP 10.96.0.15:5060;branch=z9hG4bKPj5fe38d71-f8b8-4e93-85aa-2e7af7472e1a;alias\r\nFrom: \"Anonymous\" <sip:anonymous@anonymous.invalid>;tag=2f41957b-9c9d-45d1-a18c-310ce92516ba\r\nTo: <sip:+821021656521@sip.telnyx.com>\r\nCall-ID: 1ced6b72-70c6-4c45-82e1-078568bf9d45\r\nCSeq: 2594 INVITE\r\nServer: Telnyx SIP Proxy\r\nContent-Length: 0\r\n\r\n",
				"SIP/2.0 180 Ringing\r\nVia: SIP/2.0/TCP 10.96.0.15:5060;branch=z9hG4bKPj5fe38d71-f8b8-4e93-85aa-2e7af7472e1a;alias\r\nRecord-Route: <sip:10.255.0.1;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>\r\nRecord-Route: <sip:192.76.120.10;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>\r\nRecord-Route: <sip:34.90.68.237:5060;r2=on;lr>\r\nRecord-Route: <sip:10.164.0.20;transport=tcp;r2=on;lr>\r\nFrom: \"Anonymous\" <sip:anonymous@anonymous.invalid>;tag=2f41957b-9c9d-45d1-a18c-310ce92516ba\r\nTo: <sip:+821021656521@sip.telnyx.com>;tag=2cDr76BUDp2SF\r\nCall-ID: 1ced6b72-70c6-4c45-82e1-078568bf9d45\r\nCSeq: 2594 INVITE\r\nContact: <sip:+821021656521@10.31.35.4:5070;transport=udp>\r\nAccept: application/sdp\r\nAllow: INVITE, ACK, BYE, CANCEL, OPTIONS, MESSAGE, INFO, UPDATE, REFER, NOTIFY\r\nSupported: path\r\nAllow-Events: talk, hold, conference, refer\r\nContent-Length: 0\r\n\r\n",
				"SIP/2.0 200 OK\r\nVia: SIP/2.0/TCP 10.96.0.15:5060;branch=z9hG4bKPj5fe38d71-f8b8-4e93-85aa-2e7af7472e1a;alias\r\nRecord-Route: <sip:10.255.0.1;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>\r\nRecord-Route: <sip:192.76.120.10;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>\r\nRecord-Route: <sip:34.90.68.237:5060;r2=on;lr>\r\nRecord-Route: <sip:10.164.0.20;transport=tcp;r2=on;lr>\r\nFrom: \"Anonymous\" <sip:anonymous@anonymous.invalid>;tag=2f41957b-9c9d-45d1-a18c-310ce92516ba\r\nTo: <sip:+821021656521@sip.telnyx.com>;tag=2cDr76BUDp2SF\r\nCall-ID: 1ced6b72-70c6-4c45-82e1-078568bf9d45\r\nCSeq: 2594 INVITE\r\nContact: <sip:+821021656521@10.31.35.4:5070;transport=udp>\r\nAllow: INVITE, ACK, BYE, CANCEL, OPTIONS, MESSAGE, INFO, UPDATE, REFER, NOTIFY\r\nSupported: path\r\nAllow-Events: talk, hold, conference, refer\r\nContent-Type: application/sdp\r\nContent-Disposition: session\r\nContent-Length: 284\r\n\r\nv=0\r\no=FreeSWITCH 1749742007 1749742008 IN IP4 10.164.0.12\r\ns=FreeSWITCH\r\nc=IN IP4 10.164.0.12\r\nt=0 0\r\nm=audio 23402 RTP/AVPF 0 101\r\na=rtpmap:0 PCMU/8000\r\na=rtpmap:101 telephone-event/8000\r\na=fmtp:101 0-16\r\na=silenceSupp:off - - - -\r\na=sendrecv\r\na=rtcp:23402\r\na=rtcp-mux\r\na=ptime:20\r\n",
				"ACK sip:+821021656521@10.31.35.4:5070;transport=udp SIP/2.0\r\nVia: SIP/2.0/TCP 10.96.0.15:5060;branch=z9hG4bKPj0a7ea6c1-1345-4606-bf8f-9f7ba3c41d9b;alias\r\nFrom: \"Anonymous\" <sip:anonymous@anonymous.invalid>;tag=2f41957b-9c9d-45d1-a18c-310ce92516ba\r\nTo: <sip:+821021656521@sip.telnyx.com>;tag=2cDr76BUDp2SF\r\nCall-ID: 1ced6b72-70c6-4c45-82e1-078568bf9d45\r\nCSeq: 2594 ACK\r\nRoute: <sip:10.164.0.20;transport=tcp;lr;r2=on>\r\nRoute: <sip:34.90.68.237:5060;lr;r2=on>\r\nRoute: <sip:192.76.120.10;lr;r2=on;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>\r\nRoute: <sip:10.255.0.1;lr;r2=on;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>\r\nMax-Forwards: 70\r\nUser-Agent: voipbin\r\nContent-Length:  0\r\n\r\n",
			},

			expectedRes: &recoveryDetail{
				RequestURI: "sip:+821021656521@10.31.35.4:5070;transport=udp",
				Routes:     "<sip:34.90.68.237:5060;r2=on;lr>, <sip:192.76.120.10;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>, <sip:10.255.0.1;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>",
				CallID:     "1ced6b72-70c6-4c45-82e1-078568bf9d45",

				FromDisplay: "Anonymous",
				FromURI:     "sip:anonymous@anonymous.invalid",
				FromTag:     "2f41957b-9c9d-45d1-a18c-310ce92516ba",

				ToDisplay: "",
				ToURI:     "sip:+821021656521@sip.telnyx.com",
				ToTag:     "2cDr76BUDp2SF",
				CSeq:      2595,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &recoveryHandler{
				requestHandler: mockReq,

				httpClient: &http.Client{
					Timeout: 30 * time.Second,
				},

				homerAPIAddress: "http://homer.voipbin.net",
				homerAuthToken:  "qNknsKxoWaaAbmzVzentVRhAskkuxlOVEIFUbssAlzYeQbFDFLqOsrDpYtvoiClKAxXHFXJYxpbGeJoQ",
				loadBalancerIPs: []string{"34.90.68.237"},
			}
			ctx := context.Background()

			sipMessages := []*sip.Msg{}
			for _, msg := range tt.messages {
				tmp, err := sip.ParseMsg([]byte(msg))
				if err != nil {
					t.Errorf("Wrong match. expected: ok, got: %v", err)
				}

				sipMessages = append(sipMessages, tmp)
			}

			res, err := h.getRecoveryDetail(ctx, sipMessages)
			if err != nil {
				t.Errorf("Wrong match. expected: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

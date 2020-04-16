package arihandler

import (
	"testing"

	"github.com/golang/mock/gomock"
	rabbitmq "gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

func TestSetSock(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockRabbit := rabbitmq.NewMockRabbit(mc)
	if mockRabbit == nil {
		t.Errorf("Error")
	}

	reqHandler := requestHandler{}
	reqHandler.SetSock(mockRabbit)

	if reqHandler.rabbitSock != mockRabbit {
		t.Errorf("Wrong match. expact: true, got: false")
	}
}

func TestChannelAnswer(t *testing.T) {

	type test struct {
		name       string
		asteriskID string
		channelID  string

		expectURL string
	}

	tests := []test{
		{
			"normal",
			"00:11:22:33:44:55",
			"5734c890-7f6e-11ea-9520-6f774800cd74",
			"/ari/channels/5734c890-7f6e-11ea-9520-6f774800cd74/answer",
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockRequester := NewMockRequester(mc)
	reqHandler := requestHandler{}
	reqHandler.requester = mockRequester

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockRequester.EXPECT().sendARIRequest(gomock.Any(), tt.asteriskID, tt.expectURL, reqPost, gomock.Any(), "", "").
				Return(Response{StatusCode: 200, Data: ""}, nil)

			err := reqHandler.ChannelAnswer(tt.asteriskID, tt.channelID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func TestChannelContinue(t *testing.T) {

	type test struct {
		name          string
		asteriskID    string
		channelID     string
		context       string
		extension     string
		priority      int
		label         string
		expectURL     string
		expectPayload string
	}

	tests := []test{
		{
			"have all item",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-context",
			"testcall",
			1,
			"testlabel",
			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/continue",
			`{"context":"test-context","extension":"testcall","priority":1,"label":"testlabel"}`,
		},
		{
			"has no label",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-context",
			"testcall",
			1,
			"",
			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/continue",
			`{"context":"test-context","extension":"testcall","priority":1,"label":""}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockRequester := NewMockRequester(mc)
	reqHandler := requestHandler{}
	reqHandler.requester = mockRequester

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockRequester.EXPECT().sendARIRequest(gomock.Any(), tt.asteriskID, tt.expectURL, reqPost, gomock.Any(), ContentTypeJSON, tt.expectPayload).
				Return(Response{StatusCode: 200, Data: ""}, nil)

			err := reqHandler.ChannelContinue(tt.asteriskID, tt.channelID, tt.context, tt.extension, tt.priority, tt.label)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

func TestChannelChannelVariableSet(t *testing.T) {

	type test struct {
		name          string
		asteriskID    string
		channelID     string
		variable      string
		value         string
		expectURL     string
		expectPayload string
	}

	tests := []test{
		{
			"have all item",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-variable",
			"test-value",
			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/variable",
			`{"variable":"test-variable","value":"test-value"}`,
		},
		{
			"empty value",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"test-variable",
			"",
			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/variable",
			`{"variable":"test-variable","value":""}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockRequester := NewMockRequester(mc)
	reqHandler := requestHandler{}
	reqHandler.requester = mockRequester

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockRequester.EXPECT().sendARIRequest(gomock.Any(), tt.asteriskID, tt.expectURL, reqPost, gomock.Any(), ContentTypeJSON, tt.expectPayload).
				Return(Response{StatusCode: 200, Data: ""}, nil)

			err := reqHandler.ChannelVariableSet(tt.asteriskID, tt.channelID, tt.variable, tt.value)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}

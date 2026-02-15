package pipecatframe

import (
	"testing"
)

func TestRTVIProtocolVersion(t *testing.T) {
	want := "1.0.0"
	if RTVIProtocolVersion != want {
		t.Errorf("RTVIProtocolVersion = %v, want %v", RTVIProtocolVersion, want)
	}
}

func TestRTVIMessageLabel(t *testing.T) {
	want := "rtvi-ai"
	if RTVIMessageLabel != want {
		t.Errorf("RTVIMessageLabel = %v, want %v", RTVIMessageLabel, want)
	}
}

func TestRTVIFrameTypes(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"bot-transcription", RTVIFrameTypeBotTranscription, "bot-transcription"},
		{"user-transcription", RTVIFrameTypeUserTranscription, "user-transcription"},
		{"bot-llm-text", RTVIFrameTypeBotLLMText, "bot-llm-text"},
		{"bot-llm-started", RTVIFrameTypeBotLLMStarted, "bot-llm-started"},
		{"bot-llm-stopped", RTVIFrameTypeBotLLMStopped, "bot-llm-stopped"},
		{"bot-tts-started", RTVIFrameTypeBotTTSStarted, "bot-tts-started"},
		{"bot-tts-stopped", RTVIFrameTypeBotTTSStopped, "bot-tts-stopped"},
		{"user-started-speaking", RTVIFrameTypeUserStartedSpeaking, "user-started-speaking"},
		{"user-stopped-speaking", RTVIFrameTypeUserStoppedSpeaking, "user-stopped-speaking"},
		{"bot-started-speaking", RTVIFrameTypeBotStartedSpeaking, "bot-started-speaking"},
		{"bot-stopped-speaking", RTVIFrameTypeBotStoppedSpeaking, "bot-stopped-speaking"},
		{"metrics", RTVIFrameTypeMetrics, "metrics"},
		{"user-llm-text", RTVIFrameTypeUserLLMText, "user-llm-text"},
		{"send-text", RTVIFrameTypeSendText, "send-text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("RTVIFrameType = %v, want %v", tt.constant, tt.want)
			}
		})
	}
}

func TestRTVIStructs(t *testing.T) {
	t.Run("RTVIServiceOption", func(t *testing.T) {
		opt := RTVIServiceOption{
			Name: "test-option",
			Type: "string",
		}
		if opt.Name != "test-option" {
			t.Errorf("Name = %v, want test-option", opt.Name)
		}
		if opt.Type != "string" {
			t.Errorf("Type = %v, want string", opt.Type)
		}
	})

	t.Run("RTVIService", func(t *testing.T) {
		svc := RTVIService{
			Name: "test-service",
			Options: []RTVIServiceOption{
				{Name: "opt1", Type: "bool"},
			},
		}
		if svc.Name != "test-service" {
			t.Errorf("Name = %v, want test-service", svc.Name)
		}
		if len(svc.Options) != 1 {
			t.Errorf("Options length = %v, want 1", len(svc.Options))
		}
	})

	t.Run("RTVIActionArgumentData", func(t *testing.T) {
		arg := RTVIActionArgumentData{
			Name:  "arg1",
			Value: "value1",
		}
		if arg.Name != "arg1" {
			t.Errorf("Name = %v, want arg1", arg.Name)
		}
		if arg.Value != "value1" {
			t.Errorf("Value = %v, want value1", arg.Value)
		}
	})

	t.Run("RTVIActionArgument", func(t *testing.T) {
		arg := RTVIActionArgument{
			Name: "arg1",
			Type: "string",
		}
		if arg.Name != "arg1" {
			t.Errorf("Name = %v, want arg1", arg.Name)
		}
		if arg.Type != "string" {
			t.Errorf("Type = %v, want string", arg.Type)
		}
	})

	t.Run("RTVIAction", func(t *testing.T) {
		action := RTVIAction{
			Service: "test-service",
			Action:  "test-action",
			Arguments: []RTVIActionArgument{
				{Name: "arg1", Type: "string"},
			},
			Result: "string",
		}
		if action.Service != "test-service" {
			t.Errorf("Service = %v, want test-service", action.Service)
		}
		if action.Action != "test-action" {
			t.Errorf("Action = %v, want test-action", action.Action)
		}
		if len(action.Arguments) != 1 {
			t.Errorf("Arguments length = %v, want 1", len(action.Arguments))
		}
	})

	t.Run("RTVIServiceOptionConfig", func(t *testing.T) {
		config := RTVIServiceOptionConfig{
			Name:  "opt1",
			Value: "val1",
		}
		if config.Name != "opt1" {
			t.Errorf("Name = %v, want opt1", config.Name)
		}
		if config.Value != "val1" {
			t.Errorf("Value = %v, want val1", config.Value)
		}
	})

	t.Run("RTVIServiceConfig", func(t *testing.T) {
		config := RTVIServiceConfig{
			Service: "test-service",
			Options: []RTVIServiceOptionConfig{
				{Name: "opt1", Value: "val1"},
			},
		}
		if config.Service != "test-service" {
			t.Errorf("Service = %v, want test-service", config.Service)
		}
		if len(config.Options) != 1 {
			t.Errorf("Options length = %v, want 1", len(config.Options))
		}
	})

	t.Run("RTVIConfig", func(t *testing.T) {
		config := RTVIConfig{
			Config: []RTVIServiceConfig{
				{Service: "svc1"},
			},
		}
		if len(config.Config) != 1 {
			t.Errorf("Config length = %v, want 1", len(config.Config))
		}
	})

	t.Run("RTVIUpdateConfig", func(t *testing.T) {
		config := RTVIUpdateConfig{
			Config: []RTVIServiceConfig{
				{Service: "svc1"},
			},
			Interrupt: true,
		}
		if len(config.Config) != 1 {
			t.Errorf("Config length = %v, want 1", len(config.Config))
		}
		if !config.Interrupt {
			t.Errorf("Interrupt = false, want true")
		}
	})

	t.Run("RTVIActionRunArgument", func(t *testing.T) {
		arg := RTVIActionRunArgument{
			Name:  "arg1",
			Value: "val1",
		}
		if arg.Name != "arg1" {
			t.Errorf("Name = %v, want arg1", arg.Name)
		}
		if arg.Value != "val1" {
			t.Errorf("Value = %v, want val1", arg.Value)
		}
	})

	t.Run("RTVIActionRun", func(t *testing.T) {
		run := RTVIActionRun{
			Service: "svc1",
			Action:  "act1",
			Arguments: []RTVIActionRunArgument{
				{Name: "arg1", Value: "val1"},
			},
		}
		if run.Service != "svc1" {
			t.Errorf("Service = %v, want svc1", run.Service)
		}
		if run.Action != "act1" {
			t.Errorf("Action = %v, want act1", run.Action)
		}
		if len(run.Arguments) != 1 {
			t.Errorf("Arguments length = %v, want 1", len(run.Arguments))
		}
	})

	t.Run("RTVIActionFrame", func(t *testing.T) {
		msgID := "msg-123"
		frame := RTVIActionFrame{
			RTVIActionRun: RTVIActionRun{
				Service: "svc1",
				Action:  "act1",
			},
			MessageID: &msgID,
		}
		if frame.RTVIActionRun.Service != "svc1" {
			t.Errorf("Service = %v, want svc1", frame.RTVIActionRun.Service)
		}
		if *frame.MessageID != "msg-123" {
			t.Errorf("MessageID = %v, want msg-123", *frame.MessageID)
		}
	})

	t.Run("RTVIRawClientMessageData", func(t *testing.T) {
		data := RTVIRawClientMessageData{
			Type: "test-type",
			Data: "test-data",
		}
		if data.Type != "test-type" {
			t.Errorf("Type = %v, want test-type", data.Type)
		}
		if data.Data != "test-data" {
			t.Errorf("Data = %v, want test-data", data.Data)
		}
	})

	t.Run("RTVIClientMessage", func(t *testing.T) {
		msg := RTVIClientMessage{
			MsgID: "msg-123",
			Type:  "test-type",
			Data:  "test-data",
		}
		if msg.MsgID != "msg-123" {
			t.Errorf("MsgID = %v, want msg-123", msg.MsgID)
		}
		if msg.Type != "test-type" {
			t.Errorf("Type = %v, want test-type", msg.Type)
		}
	})

	t.Run("RTVIClientMessageFrame", func(t *testing.T) {
		frame := RTVIClientMessageFrame{
			MsgID: "msg-123",
			Type:  "test-type",
			Data:  "test-data",
		}
		if frame.MsgID != "msg-123" {
			t.Errorf("MsgID = %v, want msg-123", frame.MsgID)
		}
		if frame.Type != "test-type" {
			t.Errorf("Type = %v, want test-type", frame.Type)
		}
	})

	t.Run("RTVIServerResponseFrame", func(t *testing.T) {
		errMsg := "error message"
		frame := RTVIServerResponseFrame{
			ClientMsg: RTVIClientMessageFrame{
				MsgID: "msg-123",
			},
			Data:  "response-data",
			Error: &errMsg,
		}
		if frame.ClientMsg.MsgID != "msg-123" {
			t.Errorf("ClientMsg.MsgID = %v, want msg-123", frame.ClientMsg.MsgID)
		}
		if frame.Data != "response-data" {
			t.Errorf("Data = %v, want response-data", frame.Data)
		}
		if *frame.Error != "error message" {
			t.Errorf("Error = %v, want error message", *frame.Error)
		}
	})

	t.Run("RTVIRawServerResponseData", func(t *testing.T) {
		data := RTVIRawServerResponseData{
			Type: "test-type",
			Data: "test-data",
		}
		if data.Type != "test-type" {
			t.Errorf("Type = %v, want test-type", data.Type)
		}
	})

	t.Run("RTVIServerResponse", func(t *testing.T) {
		resp := RTVIServerResponse{
			Label: RTVIMessageLabel,
			Type:  "server-response",
			ID:    "id-123",
			Data: RTVIRawServerResponseData{
				Type: "test",
			},
		}
		if resp.Label != RTVIMessageLabel {
			t.Errorf("Label = %v, want %v", resp.Label, RTVIMessageLabel)
		}
		if resp.Type != "server-response" {
			t.Errorf("Type = %v, want server-response", resp.Type)
		}
	})

	t.Run("RTVIMessage", func(t *testing.T) {
		msg := RTVIMessage{
			Label: RTVIMessageLabel,
			Type:  "test-type",
			ID:    "id-123",
			Data: map[string]interface{}{
				"key": "value",
			},
		}
		if msg.Label != RTVIMessageLabel {
			t.Errorf("Label = %v, want %v", msg.Label, RTVIMessageLabel)
		}
		if msg.Type != "test-type" {
			t.Errorf("Type = %v, want test-type", msg.Type)
		}
	})

	t.Run("RTVIErrorResponseData", func(t *testing.T) {
		data := RTVIErrorResponseData{
			Error: "test error",
		}
		if data.Error != "test error" {
			t.Errorf("Error = %v, want test error", data.Error)
		}
	})

	t.Run("RTVIErrorResponse", func(t *testing.T) {
		resp := RTVIErrorResponse{
			Label: RTVIMessageLabel,
			Type:  "error-response",
			ID:    "id-123",
			Data: RTVIErrorResponseData{
				Error: "test error",
			},
		}
		if resp.Label != RTVIMessageLabel {
			t.Errorf("Label = %v, want %v", resp.Label, RTVIMessageLabel)
		}
		if resp.Data.Error != "test error" {
			t.Errorf("Data.Error = %v, want test error", resp.Data.Error)
		}
	})

	t.Run("RTVIErrorData", func(t *testing.T) {
		data := RTVIErrorData{
			Error: "test error",
			Fatal: true,
		}
		if data.Error != "test error" {
			t.Errorf("Error = %v, want test error", data.Error)
		}
		if !data.Fatal {
			t.Errorf("Fatal = false, want true")
		}
	})

	t.Run("RTVIError", func(t *testing.T) {
		err := RTVIError{
			Label: RTVIMessageLabel,
			Type:  "error",
			Data: RTVIErrorData{
				Error: "test error",
				Fatal: true,
			},
		}
		if err.Label != RTVIMessageLabel {
			t.Errorf("Label = %v, want %v", err.Label, RTVIMessageLabel)
		}
		if err.Data.Error != "test error" {
			t.Errorf("Data.Error = %v, want test error", err.Data.Error)
		}
	})

	t.Run("RTVITextMessageData", func(t *testing.T) {
		data := RTVITextMessageData{
			Text: "test text",
		}
		if data.Text != "test text" {
			t.Errorf("Text = %v, want test text", data.Text)
		}
	})

	t.Run("RTVIBotTranscriptionMessage", func(t *testing.T) {
		msg := RTVIBotTranscriptionMessage{
			Label: RTVIMessageLabel,
			Type:  "bot-transcription",
			Data: RTVITextMessageData{
				Text: "test",
			},
		}
		if msg.Label != RTVIMessageLabel {
			t.Errorf("Label = %v, want %v", msg.Label, RTVIMessageLabel)
		}
		if msg.Data.Text != "test" {
			t.Errorf("Data.Text = %v, want test", msg.Data.Text)
		}
	})

	t.Run("RTVISendTextData", func(t *testing.T) {
		data := RTVISendTextData{
			Content: "test content",
			Options: &RTVISendTextOptions{
				RunImmediately: true,
				AudioResponse:  false,
			},
		}
		if data.Content != "test content" {
			t.Errorf("Content = %v, want test content", data.Content)
		}
		if !data.Options.RunImmediately {
			t.Errorf("Options.RunImmediately = false, want true")
		}
	})

	t.Run("RTVIBotReadyData", func(t *testing.T) {
		data := RTVIBotReadyData{
			Version: "1.0.0",
			Config: []RTVIServiceConfig{
				{Service: "svc1"},
			},
			About: map[string]interface{}{
				"key": "value",
			},
		}
		if data.Version != "1.0.0" {
			t.Errorf("Version = %v, want 1.0.0", data.Version)
		}
		if len(data.Config) != 1 {
			t.Errorf("Config length = %v, want 1", len(data.Config))
		}
	})

	t.Run("RTVILLMFunctionCallMessageData", func(t *testing.T) {
		data := RTVILLMFunctionCallMessageData{
			FunctionName: "test_func",
			ToolCallID:   "tool-123",
			Args: map[string]interface{}{
				"arg1": "val1",
			},
		}
		if data.FunctionName != "test_func" {
			t.Errorf("FunctionName = %v, want test_func", data.FunctionName)
		}
		if data.ToolCallID != "tool-123" {
			t.Errorf("ToolCallID = %v, want tool-123", data.ToolCallID)
		}
	})

	t.Run("RTVIObserverParams", func(t *testing.T) {
		enabled := true
		params := RTVIObserverParams{
			BotLLMEnabled:            true,
			BotTTSEnabled:            true,
			BotSpeakingEnabled:       true,
			BotAudioLevelEnabled:     true,
			UserLLMEnabled:           true,
			UserSpeakingEnabled:      true,
			UserTranscriptionEnabled: true,
			UserAudioLevelEnabled:    true,
			MetricsEnabled:           true,
			SystemLogsEnabled:        true,
			ErrorsEnabled:            &enabled,
			AudioLevelPeriodSecs:     0.5,
		}
		if !params.BotLLMEnabled {
			t.Errorf("BotLLMEnabled = false, want true")
		}
		if params.AudioLevelPeriodSecs != 0.5 {
			t.Errorf("AudioLevelPeriodSecs = %v, want 0.5", params.AudioLevelPeriodSecs)
		}
	})
}

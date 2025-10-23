package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_pythonrunner.go -source pythonrunner.go -build_flags=-mod=mod

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	defaultPipecatRunnerListenAddress = "http://localhost:8000/run"
)

type pythonRunner struct {
}

type PythonRunner interface {
	Start(ctx context.Context, pipecatcallID uuid.UUID, uri string, llm string, stt string, tts string, voiceID string, messages []map[string]any) error
}

func NewPythonRunner() PythonRunner {
	return &pythonRunner{}
}

func (h *pythonRunner) Start(ctx context.Context, pipecatcallID uuid.UUID, uri string, llm string, stt string, tts string, voiceID string, messages []map[string]any) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Start",
	})

	type PipelineRequest struct {
		ID          uuid.UUID        `json:"id,omitempty"`
		WsServerURL string           `json:"ws_server_url,omitempty"`
		LLM         string           `json:"llm,omitempty"`
		TTS         string           `json:"tts,omitempty"`
		STT         string           `json:"stt,omitempty"`
		VoiceID     string           `json:"voice_id,omitempty"`
		Messages    []map[string]any `json:"messages,omitempty"`
	}

	reqBody := PipelineRequest{
		ID:          pipecatcallID,
		WsServerURL: uri,
		LLM:         llm,
		STT:         stt,
		TTS:         tts,
		VoiceID:     voiceID,
		Messages:    messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrapf(err, "could not marshal request body for python runner")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", defaultPipecatRunnerListenAddress, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrapf(err, "could not create request for python runner")
	}
	req.Header.Set("Content-Type", "application/json")

	log.WithField("request_body", string(jsonData)).Debugf("Sending request to python runner")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "could not send request to python runner")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "could not read response body from python runner")
	}

	log.Debugf("Response status: %s", resp.Status)
	log.Debugf("Response body: %v", body)

	return nil
}

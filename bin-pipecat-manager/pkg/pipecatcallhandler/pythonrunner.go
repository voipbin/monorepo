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
	defaultPipecatRunnerListenAddress = "http://localhost:8000"
)

type pythonRunner struct {
}

type PythonRunner interface {
	Start(
		ctx context.Context,
		pipecatcallID uuid.UUID,
		llmType string,
		llmKey string,
		stt string,
		tts string,
		voiceID string,
		messages []map[string]any,
	) error
	Stop(ctx context.Context, pipecatcallID uuid.UUID) error
}

func NewPythonRunner() PythonRunner {
	return &pythonRunner{}
}

func (h *pythonRunner) Start(
	ctx context.Context,
	pipecatcallID uuid.UUID,
	llmType string,
	llmKey string,
	stt string,
	tts string,
	voiceID string,
	messages []map[string]any,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Start",
	})

	// // only used to send data to the python runner
	reqBody := struct {
		ID       uuid.UUID        `json:"id,omitempty"`
		LLMType  string           `json:"llm_type,omitempty"`
		LLMKey   string           `json:"llm_key,omitempty"`
		TTS      string           `json:"tts,omitempty"`
		STT      string           `json:"stt,omitempty"`
		VoiceID  string           `json:"voice_id,omitempty"`
		Messages []map[string]any `json:"messages,omitempty"`
	}{
		ID:       pipecatcallID,
		LLMType:  llmType,
		LLMKey:   llmKey,
		STT:      stt,
		TTS:      tts,
		VoiceID:  voiceID,
		Messages: messages,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrapf(err, "could not marshal request body for python runner")
	}

	url := defaultPipecatRunnerListenAddress + "/run"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
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

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("python runner returned non-200 status: %s, body: %s", resp.Status, string(body))
	}

	return nil
}

func (h *pythonRunner) Stop(ctx context.Context, pipecatcallID uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Stop",
		"pipecatcall_id": pipecatcallID,
	})

	url := defaultPipecatRunnerListenAddress + "/stop?id=" + pipecatcallID.String()
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return errors.Wrapf(err, "could not create stop request for python runner")
	}

	log.Debugf("Sending stop request to python runner")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "could not send stop request to python runner")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "could not read stop response body from python runner")
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("python runner returned non-200 status on stop: %s, body: %s", resp.Status, string(body))
	}

	return nil
}

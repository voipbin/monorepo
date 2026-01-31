package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_pythonrunner.go -source pythonrunner.go -build_flags=-mod=mod

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	aitool "monorepo/bin-ai-manager/models/tool"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	defaultPipecatRunnerListenAddress = "http://localhost:8000"
)

// httpClient is a package-level HTTP client with connection pooling.
// Reusing connections avoids TCP handshake overhead for each request.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

type pythonRunner struct {
}

type PythonRunner interface {
	Start(
		ctx context.Context,
		pipecatcallID uuid.UUID,
		llmType string,
		llmKey string,
		llmMessages []map[string]any,
		sttType string,
		sttLanguage string,
		ttsType string,
		ttsLanguage string,
		ttsVoiceID string,
		tools []aitool.Tool,
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
	llmMessages []map[string]any,
	sttType string,
	sttLanguage string,
	ttsType string,
	ttsLanguage string,
	ttsVoiceID string,
	tools []aitool.Tool,
) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Start",
	})

	// Request body structure for Python runner
	reqBody := struct {
		ID          uuid.UUID        `json:"id,omitempty"`
		LLMType     string           `json:"llm_type,omitempty"`
		LLMKey      string           `json:"llm_key,omitempty"`
		LLMMessages []map[string]any `json:"llm_messages,omitempty"`
		STTType     string           `json:"stt_type,omitempty"`
		STTLanguage string           `json:"stt_language,omitempty"`
		TTSType     string           `json:"tts_type,omitempty"`
		TTSLanguage string           `json:"tts_language,omitempty"`
		TTSVoiceID  string           `json:"tts_voice_id,omitempty"`
		Tools       []aitool.Tool    `json:"tools,omitempty"`
	}{
		ID:          pipecatcallID,
		LLMType:     llmType,
		LLMKey:      llmKey,
		LLMMessages: llmMessages,
		STTType:     sttType,
		STTLanguage: sttLanguage,
		TTSType:     ttsType,
		TTSLanguage: ttsLanguage,
		TTSVoiceID:  ttsVoiceID,
		Tools:       tools,
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
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "could not send request to python runner")
	}
	defer func() {
		// Drain and close body to enable connection reuse
		_, _ = io.Copy(io.Discard, resp.Body)
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
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "could not send stop request to python runner")
	}
	defer func() {
		// Drain and close body to enable connection reuse
		_, _ = io.Copy(io.Discard, resp.Body)
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

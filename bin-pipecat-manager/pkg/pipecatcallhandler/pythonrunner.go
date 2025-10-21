package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_pythonrunner.go -source pythonrunner.go -build_flags=-mod=mod

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	defaultPipecatRunnerListenAddress = "http://localhost:8000/run"
)

type pythonRunner struct {
}

type PythonRunner interface {
	// Start(interpreter string, args []string) (*exec.Cmd, error)
	Start(ctx context.Context, uri string, llm string, stt string, tts string, voiceID string, messages []map[string]any) error
}

func NewPythonRunner() PythonRunner {
	return &pythonRunner{}
}

// func (h *pythonRunner) Start(interpreter string, args []string) (*exec.Cmd, error) {
// 	log := logrus.WithFields(logrus.Fields{
// 		"func":        "Start",
// 		"interpreter": interpreter,
// 		"args":        args,
// 	})

// 	cmd := exec.Command(interpreter, args...)

// 	stdoutPipe, err := cmd.StdoutPipe()
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not get stdout pipe for python")
// 	}
// 	stderrPipe, err := cmd.StderrPipe()
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "could not get stderr pipe for python")
// 	}

// 	log.Debugf("Executing pipecat python script...")
// 	if errStart := cmd.Start(); errStart != nil {
// 		return nil, errors.Wrapf(errStart, "could not start python client")
// 	}
// 	log.Debugf("Python client process started with PID: %d", cmd.Process.Pid)

// 	go func() {
// 		scanner := bufio.NewScanner(stdoutPipe)
// 		for scanner.Scan() {
// 			log.Debugf("[PYTHON-CLIENT-STDOUT] %s\n", scanner.Text())
// 		}
// 		if err := scanner.Err(); err != nil {
// 			log.Errorf("Error reading Python client stdout: %v", err)
// 		}
// 	}()

// 	go func() {
// 		scanner := bufio.NewScanner(stderrPipe)
// 		for scanner.Scan() {
// 			log.Debugf("[PYTHON-CLIENT-STDERR] %s\n", scanner.Text())
// 		}
// 		if err := scanner.Err(); err != nil {
// 			log.Errorf("Error reading Python client stderr: %v", err)
// 		}
// 	}()

// 	go func() {
// 		// wait for the python process to exit
// 		if errPython := cmd.Wait(); errPython != nil {
// 			log.Errorf("Python client process exited with error: %v", errPython)
// 		}
// 	}()

// 	return cmd, nil
// }

func (h *pythonRunner) Start(ctx context.Context, uri string, llm string, stt string, tts string, voiceID string, messages []map[string]any) error {
	log := logrus.WithFields(logrus.Fields{
		"func": "Start",
	})

	type PipelineRequest struct {
		WsServerURL string           `json:"ws_server_url,omitempty"`
		LLM         string           `json:"llm,omitempty"`
		TTS         string           `json:"tts,omitempty"`
		STT         string           `json:"stt,omitempty"`
		VoiceID     string           `json:"voice_id,omitempty"`
		Messages    []map[string]any `json:"messages,omitempty"`
	}

	reqBody := PipelineRequest{
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

	body, _ := io.ReadAll(resp.Body)
	log.Debugf("Response status: %s", resp.Status)
	log.Debugf("Response body: %v", body)

	return nil
}

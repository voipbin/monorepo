package pipecatcallhandler

//go:generate mockgen -package pipecatcallhandler -destination ./mock_pythonrunner.go -source pythonrunner.go -build_flags=-mod=mod

import (
	"bufio"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type pythonRunner struct {
}

type PythonRunner interface {
	Start(interpreter string, args []string) (*exec.Cmd, error)
}

func NewPythonRunner() PythonRunner {
	return &pythonRunner{}
}

func (h *pythonRunner) Start(interpreter string, args []string) (*exec.Cmd, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "Start",
		"interpreter": interpreter,
		"args":        args,
	})

	cmd := exec.Command(interpreter, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrapf(err, "could to get stdout pipe for python")
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.Wrapf(err, "could to get stderr pipe for python")
	}

	log.Debugf("Executing pipecat python script...")
	if errStart := cmd.Start(); errStart != nil {
		return nil, errors.Wrapf(errStart, "could not start python client")
	}
	log.Debugf("Python client process started with PID: %d", cmd.Process.Pid)

	go func() {
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			log.Debugf("[PYTHON-CLIENT-STDOUT] %s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Errorf("Error reading Python client stdout: %v", err)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			log.Debugf("[PYTHON-CLIENT-STDERR] %s\n", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Errorf("Error reading Python client stderr: %v", err)
		}
	}()

	go func() {
		// wait for the python process to exit
		if errPython := cmd.Wait(); errPython != nil {
			log.Errorf("Python client process exited with error: %v", errPython)
		}
	}()

	return cmd, nil
}

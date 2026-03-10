package processmanager

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/voip-rtpengine-proxy/pkg/gcsuploader"
)

const defaultMaxConcurrent = 20

type trackedProcess struct {
	cmd      *exec.Cmd
	pcapPath string
	timer    *time.Timer
	waitOnce sync.Once
	waitErr  error
}

type manager struct {
	mu            sync.Mutex
	processes     map[string]*trackedProcess
	maxConcurrent int
	interfaceName string
	safetyTimeout time.Duration
	uploader      gcsuploader.Uploader
}

// NewManager creates a process manager.
func NewManager(interfaceName string, safetyTimeout time.Duration, uploader gcsuploader.Uploader) ProcessManager {
	return &manager{
		processes:     make(map[string]*trackedProcess),
		maxConcurrent: defaultMaxConcurrent,
		interfaceName: interfaceName,
		safetyTimeout: safetyTimeout,
		uploader:      uploader,
	}
}

// Count returns the number of running processes.
func (m *manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.processes)
}

// Exec starts a new process. The command must be whitelisted and parameters sanitized.
func (m *manager) Exec(id, command string, parameters []string) error {
	log := logrus.WithFields(logrus.Fields{"func": "Exec", "id": id, "command": command})

	if err := validateCommand(command); err != nil {
		return fmt.Errorf("validate command: %w", err)
	}

	if err := sanitizeParameters(parameters); err != nil {
		return fmt.Errorf("sanitize parameters: %w", err)
	}

	// Reject any -w in incoming parameters; the proxy always constructs the write path.
	for _, p := range parameters {
		if p == "-w" {
			return fmt.Errorf("parameter -w is not allowed (write path is managed internally)")
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.processes[id]; exists {
		return fmt.Errorf("process with id %q already running", id)
	}

	if len(m.processes) >= m.maxConcurrent {
		return fmt.Errorf("max concurrent captures reached (%d)", m.maxConcurrent)
	}

	pcapPath := fmt.Sprintf("/tmp/%s.pcap", id)
	args := []string{"-i", m.interfaceName, "-s", "0", "-w", pcapPath}
	args = append(args, parameters...)

	if err := validateWritePath(args); err != nil {
		return fmt.Errorf("validate write path: %w", err)
	}

	cmd := exec.Command(command, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", command, err)
	}

	log.WithField("pid", cmd.Process.Pid).Infof("Started process. pcap: %s", pcapPath)

	timer := time.AfterFunc(m.safetyTimeout, func() {
		log.Warnf("Safety timeout reached (%v). Auto-killing process.", m.safetyTimeout)
		if _, err := m.killAndUpload(id); err != nil {
			log.WithError(err).Errorf("Safety timeout kill failed for %s", id)
		}
	})

	m.processes[id] = &trackedProcess{
		cmd:      cmd,
		pcapPath: pcapPath,
		timer:    timer,
	}

	return nil
}

// Kill stops a running process by ID.
func (m *manager) Kill(id string) (string, error) {
	return m.killAndUpload(id)
}

func (m *manager) killAndUpload(id string) (string, error) {
	log := logrus.WithFields(logrus.Fields{"func": "killAndUpload", "id": id})

	m.mu.Lock()
	tracked, exists := m.processes[id]
	if !exists {
		m.mu.Unlock()
		return "", fmt.Errorf("no process with id %q", id)
	}
	tracked.timer.Stop()
	delete(m.processes, id)
	m.mu.Unlock()

	if tracked.cmd.Process != nil {
		log.Infof("Sending SIGTERM to pid %d", tracked.cmd.Process.Pid)
		tracked.cmd.Process.Signal(syscall.SIGTERM)
	}

	done := make(chan struct{})
	go func() {
		tracked.waitOnce.Do(func() {
			tracked.waitErr = tracked.cmd.Wait()
		})
		close(done)
	}()

	select {
	case <-done:
		log.Debugf("Process exited")
	case <-time.After(5 * time.Second):
		log.Warnf("Process did not exit after SIGTERM, sending SIGKILL")
		tracked.cmd.Process.Kill()
		<-done
	}

	uploaded := false
	if m.uploader != nil {
		remoteName := fmt.Sprintf("rtp-recordings/%s.pcap", id)
		log.Infof("Uploading %s to %s", tracked.pcapPath, remoteName)

		if _, err := m.uploader.Upload(tracked.pcapPath, remoteName); err != nil {
			log.Errorf("GCS upload failed: %v. Retrying once.", err)
			if _, err := m.uploader.Upload(tracked.pcapPath, remoteName); err != nil {
				log.Errorf("GCS upload retry failed: %v. Keeping local file for manual recovery.", err)
			} else {
				uploaded = true
			}
		} else {
			uploaded = true
		}
	}

	if uploaded || m.uploader == nil {
		if err := os.Remove(tracked.pcapPath); err != nil && !os.IsNotExist(err) {
			log.Warnf("Could not delete local pcap: %v", err)
		}
	}

	return tracked.pcapPath, nil
}

// Shutdown kills all running processes and uploads their pcap files.
func (m *manager) Shutdown() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.processes))
	for id := range m.processes {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		logrus.Infof("Shutdown: killing process %s", id)
		if _, err := m.killAndUpload(id); err != nil {
			logrus.Warnf("Shutdown: kill %s: %v", id, err)
		}
	}
}

// CleanOrphans removes any leftover /tmp/<uuid>.pcap files from a previous run.
// Only removes files matching the UUID naming pattern used by this process manager.
func (m *manager) CleanOrphans() {
	matches, err := filepath.Glob("/tmp/*.pcap")
	if err != nil {
		logrus.Errorf("CleanOrphans glob error: %v", err)
		return
	}
	for _, f := range matches {
		if !uuidPattern.MatchString(filepath.Base(f)) {
			continue
		}
		logrus.Infof("Cleaning orphaned pcap: %s", f)
		os.Remove(f)
	}
}

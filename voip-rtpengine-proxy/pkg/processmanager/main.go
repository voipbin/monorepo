package processmanager

//go:generate mockgen -source=main.go -destination=mock_main.go -package=processmanager

// ProcessManager manages tcpdump processes for RTP capture.
type ProcessManager interface {
	// Exec starts a new tcpdump process with the given BPF filter parameters.
	Exec(id, command string, parameters []string) error

	// Kill stops a running process and uploads its pcap to GCS.
	Kill(id string) (string, error)

	// Shutdown kills all running processes and uploads their pcap files.
	Shutdown()

	// CleanOrphans removes any leftover /tmp/*.pcap files from a previous run.
	CleanOrphans()

	// Count returns the number of running processes.
	Count() int
}

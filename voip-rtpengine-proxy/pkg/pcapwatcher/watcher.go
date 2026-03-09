package pcapwatcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"

	"monorepo/voip-rtpengine-proxy/pkg/gcsuploader"
)

const rescanInterval = 5 * time.Minute

type watcher struct {
	recordingDir string
	uploader     gcsuploader.Uploader
}

// New creates a new pcap watcher.
func New(recordingDir string, uploader gcsuploader.Uploader) Watcher {
	return &watcher{
		recordingDir: recordingDir,
		uploader:     uploader,
	}
}

func (w *watcher) Run(ctx context.Context) error {
	metadataDir := filepath.Join(w.recordingDir, "metadata")

	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("could not create metadata dir: %w", err)
	}

	// Startup scan for unprocessed files
	w.scanAndProcess(metadataDir)

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("could not create fsnotify watcher: %w", err)
	}
	defer fsWatcher.Close()

	if err := fsWatcher.Add(metadataDir); err != nil {
		return fmt.Errorf("could not watch metadata dir: %w", err)
	}

	log.WithField("dir", metadataDir).Info("pcap watcher started")

	rescanTicker := time.NewTicker(rescanInterval)
	defer rescanTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("pcap watcher stopping")
			return nil

		case event, ok := <-fsWatcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) {
				log.WithField("file", event.Name).Debug("new metadata file detected")
				if err := w.processMetadataFile(event.Name); err != nil {
					log.WithError(err).WithField("file", event.Name).Warn("failed to process metadata file")
				}
			}

		case err, ok := <-fsWatcher.Errors:
			if !ok {
				return nil
			}
			log.WithError(err).Warn("fsnotify error")

		case <-rescanTicker.C:
			w.scanAndProcess(metadataDir)
		}
	}
}

func (w *watcher) scanAndProcess(metadataDir string) {
	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		log.WithError(err).Warn("could not scan metadata dir")
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(metadataDir, name)

		if strings.HasSuffix(name, ".processing") {
			if err := w.processProcessingFile(path); err != nil {
				log.WithError(err).WithField("file", name).Warn("failed to retry processing file")
			}
			continue
		}

		if err := w.processMetadataFile(path); err != nil {
			log.WithError(err).WithField("file", name).Warn("failed to process metadata file on scan")
		}
	}
}

func (w *watcher) processMetadataFile(metadataPath string) error {
	processingPath := metadataPath + ".processing"
	if err := os.Rename(metadataPath, processingPath); err != nil {
		return fmt.Errorf("could not rename to .processing: %w", err)
	}

	return w.processProcessingFile(processingPath)
}

func (w *watcher) processProcessingFile(processingPath string) error {
	content, err := os.ReadFile(processingPath)
	if err != nil {
		return fmt.Errorf("could not read metadata file: %w", err)
	}

	baseName := filepath.Base(processingPath)
	baseName = strings.TrimSuffix(baseName, ".processing")
	callID := extractCallIDFromFilename(baseName)

	pcapPath, err := parsePcapPathFromMetadata(string(content))
	if err != nil {
		os.Remove(processingPath)
		return fmt.Errorf("could not parse metadata: %w", err)
	}

	pcapFilename := filepath.Base(pcapPath)
	objectPath := buildObjectPath(callID, pcapFilename)

	if _, err := w.uploader.Upload(pcapPath, objectPath); err != nil {
		return fmt.Errorf("could not upload pcap: %w", err)
	}

	if err := os.Remove(pcapPath); err != nil && !os.IsNotExist(err) {
		log.WithError(err).WithField("file", pcapPath).Warn("could not delete pcap file")
	}

	if err := os.Remove(processingPath); err != nil && !os.IsNotExist(err) {
		log.WithError(err).WithField("file", processingPath).Warn("could not delete processing file")
	}

	log.WithFields(log.Fields{
		"call_id":  callID,
		"pcap":     pcapFilename,
		"gcs_path": objectPath,
	}).Info("recording processed and uploaded")

	return nil
}

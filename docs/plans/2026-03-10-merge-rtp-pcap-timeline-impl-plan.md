# Merge RTP PCAP into Timeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extend timeline-manager's pcap endpoint to merge RTP pcap files from GCS with existing SIP/RTCP pcaps from Homer.

**Architecture:** Add a GCSReader interface to siphandler, fetch RTP pcap files from GCS by SIP Call-ID prefix, download to temp files, and merge all sources using a streaming k-way merge sorted by timestamp. Graceful degradation: GCS failures never break existing SIP+RTCP behavior.

**Tech Stack:** Go, google/gopacket (pcapgo), cloud.google.com/go/storage, gomock

**Design doc:** `docs/plans/2026-03-10-merge-rtp-pcap-timeline-design.md`

**Worktree:** `~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline`

**Service directory:** `bin-timeline-manager` (within the worktree)

---

### Task 1: Add GCS_BUCKET_NAME to config

**Files:**
- Modify: `bin-timeline-manager/internal/config/main.go`
- Modify: `bin-timeline-manager/internal/config/main_test.go`

**Step 1: Update config struct and bindings**

In `internal/config/main.go`:

Add `GCSBucketName` field to the `Config` struct (after `HomerAuthToken`):

```go
type Config struct {
	RabbitMQAddress         string
	PrometheusEndpoint      string
	PrometheusListenAddress string
	ClickHouseAddress       string
	ClickHouseDatabase      string
	MigrationsPath          string
	HomerAPIAddress         string
	HomerAuthToken          string
	GCSBucketName           string
}
```

Add flag in `bindConfig` (after the `homer_auth_token` line):

```go
f.String("gcs_bucket_name", "", "GCS bucket for RTP pcap recordings")
```

Add binding in the `bindings` map:

```go
"gcs_bucket_name": "GCS_BUCKET_NAME",
```

Add field in `LoadGlobalConfig`:

```go
GCSBucketName: viper.GetString("gcs_bucket_name"),
```

**Step 2: Update config tests**

In `internal/config/main_test.go`:

Add to `TestConfig_DefaultValues`:
```go
if cfg.GCSBucketName != "" {
	t.Errorf("GCSBucketName = %q, want empty", cfg.GCSBucketName)
}
```

Add to `TestConfig_AllFields` struct literal:
```go
GCSBucketName: "my-bucket",
```

Add assertion:
```go
if cfg.GCSBucketName != "my-bucket" {
	t.Errorf("GCSBucketName = %q, want %q", cfg.GCSBucketName, "my-bucket")
}
```

Add `"gcs_bucket_name"` to the flags list in `TestBindConfig`.

**Step 3: Run tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline/bin-timeline-manager
go test ./internal/config/...
```

Expected: PASS

**Step 4: Commit**

```bash
git add internal/config/main.go internal/config/main_test.go
git commit -m "NOJIRA-Merge-rtp-pcap-timeline

- bin-timeline-manager: Add GCS_BUCKET_NAME config field for RTP pcap bucket"
```

---

### Task 2: Create GCSReader interface and mock

**Files:**
- Create: `bin-timeline-manager/pkg/siphandler/gcsreader.go`

**Step 1: Write the GCSReader interface**

Create `pkg/siphandler/gcsreader.go`:

```go
package siphandler

//go:generate mockgen -package siphandler -destination ./mock_gcsreader.go -source gcsreader.go -build_flags=-mod=mod

import (
	"context"
	"io"
)

// GCSReader provides read access to GCS objects for fetching RTP pcap files.
type GCSReader interface {
	// ListObjects lists object names in the bucket matching the given prefix.
	ListObjects(ctx context.Context, prefix string) ([]string, error)

	// DownloadObject downloads a GCS object and writes its content to dest.
	DownloadObject(ctx context.Context, objectPath string, dest io.Writer) error
}
```

**Step 2: Generate mock**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline/bin-timeline-manager
go generate ./pkg/siphandler/...
```

Expected: Creates `pkg/siphandler/mock_gcsreader.go`

**Step 3: Verify it compiles**

```bash
go build ./pkg/siphandler/...
```

Expected: SUCCESS

**Step 4: Commit**

```bash
git add pkg/siphandler/gcsreader.go pkg/siphandler/mock_gcsreader.go pkg/siphandler/mock_main.go
git commit -m "NOJIRA-Merge-rtp-pcap-timeline

- bin-timeline-manager: Add GCSReader interface and generated mock for GCS pcap access"
```

---

### Task 3: Implement GCSReader using storage.Client

**Files:**
- Create: `bin-timeline-manager/pkg/siphandler/gcsreader_impl.go`
- Modify: `bin-timeline-manager/go.mod` (add cloud.google.com/go/storage dependency)

**Step 1: Add GCS dependency**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline/bin-timeline-manager
go get cloud.google.com/go/storage
```

**Step 2: Write the implementation**

Create `pkg/siphandler/gcsreader_impl.go`:

```go
package siphandler

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

type gcsReaderImpl struct {
	client     *storage.Client
	bucketName string
}

// NewGCSReader creates a GCSReader backed by a real GCS storage client.
func NewGCSReader(client *storage.Client, bucketName string) GCSReader {
	return &gcsReaderImpl{
		client:     client,
		bucketName: bucketName,
	}
}

func (g *gcsReaderImpl) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":   "ListObjects",
		"bucket": g.bucketName,
		"prefix": prefix,
	})

	it := g.client.Bucket(g.bucketName).Objects(ctx, &storage.Query{Prefix: prefix})

	names := []string{}
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not list GCS objects with prefix %s: %w", prefix, err)
		}
		names = append(names, attrs.Name)
	}

	log.WithField("count", len(names)).Debugf("Listed GCS objects. prefix: %s", prefix)
	return names, nil
}

func (g *gcsReaderImpl) DownloadObject(ctx context.Context, objectPath string, dest io.Writer) error {
	log := logrus.WithFields(logrus.Fields{
		"func":        "DownloadObject",
		"bucket":      g.bucketName,
		"object_path": objectPath,
	})

	reader, err := g.client.Bucket(g.bucketName).Object(objectPath).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("could not open GCS object %s: %w", objectPath, err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			log.WithError(closeErr).Warn("could not close GCS reader")
		}
	}()

	n, err := io.Copy(dest, reader)
	if err != nil {
		return fmt.Errorf("could not download GCS object %s: %w", objectPath, err)
	}

	log.WithField("bytes", n).Debugf("Downloaded GCS object. object_path: %s", objectPath)
	return nil
}
```

**Step 3: Tidy and vendor**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline/bin-timeline-manager
go mod tidy && go mod vendor
```

**Step 4: Verify it compiles**

```bash
go build ./pkg/siphandler/...
```

Expected: SUCCESS

**Step 5: Commit**

```bash
git add pkg/siphandler/gcsreader_impl.go go.mod go.sum
git commit -m "NOJIRA-Merge-rtp-pcap-timeline

- bin-timeline-manager: Implement GCSReader with storage.Client for listing and downloading pcap files"
```

---

### Task 4: Write mergeMultiplePcaps with tests (TDD)

**Files:**
- Modify: `bin-timeline-manager/pkg/siphandler/main.go`
- Modify: `bin-timeline-manager/pkg/siphandler/main_test.go`

**Step 1: Write failing tests for mergeMultiplePcaps**

Add to `main_test.go`. Use the existing `createTimestampedPcap` helper pattern from the file:

```go
func TestMergeMultiplePcaps(t *testing.T) {
	createTimestampedPcap := func(ts time.Time, snaplen uint32) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(snaplen, layers.LinkTypeEthernet)

		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte{10, 0, 0, 1},
			DstIP:    []byte{10, 0, 0, 2},
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload([]byte("test")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{
			Timestamp:     ts,
			CaptureLength: len(packetData),
			Length:        len(packetData),
		}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	t.Run("zero sources returns empty pcap", func(t *testing.T) {
		result, err := mergeMultiplePcaps(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result, got %d bytes", len(result))
		}
	})

	t.Run("single source passthrough", func(t *testing.T) {
		ts := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)
		pcap := createTimestampedPcap(ts, 65536)

		result, err := mergeMultiplePcaps([]io.Reader{bytes.NewReader(pcap)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		count := 0
		for {
			_, _, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			count++
		}
		if count != 1 {
			t.Errorf("expected 1 packet, got %d", count)
		}
	})

	t.Run("three sources sorted by timestamp", func(t *testing.T) {
		ts1 := time.Date(2026, 1, 1, 0, 0, 3, 0, time.UTC)
		ts2 := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)
		ts3 := time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC)

		sources := []io.Reader{
			bytes.NewReader(createTimestampedPcap(ts1, 65536)),
			bytes.NewReader(createTimestampedPcap(ts2, 65536)),
			bytes.NewReader(createTimestampedPcap(ts3, 65536)),
		}

		result, err := mergeMultiplePcaps(sources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		var timestamps []time.Time
		for {
			_, ci, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			timestamps = append(timestamps, ci.Timestamp)
		}

		if len(timestamps) != 3 {
			t.Fatalf("expected 3 packets, got %d", len(timestamps))
		}
		for i := 1; i < len(timestamps); i++ {
			if timestamps[i].Before(timestamps[i-1]) {
				t.Errorf("packets not sorted: %v before %v at index %d", timestamps[i], timestamps[i-1], i)
			}
		}
	})

	t.Run("uses max snaplen across sources", func(t *testing.T) {
		ts := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)
		sources := []io.Reader{
			bytes.NewReader(createTimestampedPcap(ts, 1500)),
			bytes.NewReader(createTimestampedPcap(ts, 65536)),
		}

		result, err := mergeMultiplePcaps(sources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		if reader.Snaplen() != 65536 {
			t.Errorf("snaplen = %d, want 65536", reader.Snaplen())
		}
	})

	t.Run("link type mismatch excludes mismatched source", func(t *testing.T) {
		ts1 := time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)
		ts2 := time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC)

		pcapEthernet := createTimestampedPcap(ts1, 65536)

		// Create a pcap with LinkTypeRaw
		var rawBuf bytes.Buffer
		rawWriter := pcapgo.NewWriter(&rawBuf)
		_ = rawWriter.WriteFileHeader(65536, layers.LinkTypeRaw)
		rawPcap := rawBuf.Bytes()

		sources := []io.Reader{
			bytes.NewReader(pcapEthernet),
			bytes.NewReader(rawPcap),
			bytes.NewReader(createTimestampedPcap(ts2, 65536)),
		}

		result, err := mergeMultiplePcaps(sources)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		count := 0
		for {
			_, _, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			count++
		}
		// Should have 2 packets (from matching Ethernet sources), raw source excluded
		if count != 2 {
			t.Errorf("expected 2 packets (mismatched excluded), got %d", count)
		}
	})
}
```

**Step 2: Run tests to verify they fail**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline/bin-timeline-manager
go test ./pkg/siphandler/... -run TestMergeMultiplePcaps -v
```

Expected: FAIL — `mergeMultiplePcaps` undefined

**Step 3: Implement mergeMultiplePcaps**

Add to `pkg/siphandler/main.go`. Add `"io"` to imports:

```go
// readerEntry tracks a pcap reader and its currently buffered packet.
type readerEntry struct {
	reader   *pcapgo.Reader
	ci       gopacket.CaptureInfo
	data     []byte
	done     bool
	linkType layers.LinkType
	snaplen  uint32
}

// mergeMultiplePcaps merges N pcap sources into a single pcap sorted by timestamp.
// Sources with mismatched link-layer types are excluded with a warning.
// Uses max(snaplen) across all included sources.
func mergeMultiplePcaps(sources []io.Reader) ([]byte, error) {
	if len(sources) == 0 {
		return []byte{}, nil
	}

	log := logrus.WithField("func", "mergeMultiplePcaps")

	// Initialize readers and read first packet from each
	var entries []*readerEntry
	for i, src := range sources {
		reader, err := pcapgo.NewReader(src)
		if err != nil {
			log.WithField("source_index", i).Warnf("Could not open pcap source, skipping: %v", err)
			continue
		}

		entry := &readerEntry{
			reader:   reader,
			linkType: reader.LinkType(),
			snaplen:  reader.Snaplen(),
		}

		// Read first packet
		data, ci, err := reader.ReadPacketData()
		if err != nil {
			entry.done = true
		} else {
			entry.ci = ci
			entry.data = data
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return []byte{}, nil
	}

	// Determine primary link type from first entry
	primaryLinkType := entries[0].linkType

	// Filter entries by link type, track max snaplen
	var included []*readerEntry
	var maxSnaplen uint32
	for _, e := range entries {
		if e.linkType != primaryLinkType {
			log.WithFields(logrus.Fields{
				"expected": primaryLinkType,
				"actual":   e.linkType,
			}).Warn("Link type mismatch, excluding source from merge.")
			continue
		}
		included = append(included, e)
		if e.snaplen > maxSnaplen {
			maxSnaplen = e.snaplen
		}
	}

	if len(included) == 0 {
		return []byte{}, nil
	}

	// Write merged output
	var buf bytes.Buffer
	writer := pcapgo.NewWriter(&buf)
	if err := writer.WriteFileHeader(maxSnaplen, primaryLinkType); err != nil {
		return nil, fmt.Errorf("could not write pcap header: %w", err)
	}

	// K-way merge: pick earliest timestamp each iteration
	for {
		// Find entry with earliest timestamp among non-done entries
		minIdx := -1
		for i, e := range included {
			if e.done {
				continue
			}
			if minIdx == -1 || e.ci.Timestamp.Before(included[minIdx].ci.Timestamp) {
				minIdx = i
			}
		}

		if minIdx == -1 {
			break // All sources exhausted
		}

		// Write the earliest packet
		if err := writer.WritePacket(included[minIdx].ci, included[minIdx].data); err != nil {
			return nil, fmt.Errorf("could not write packet: %w", err)
		}

		// Advance that reader
		data, ci, err := included[minIdx].reader.ReadPacketData()
		if err != nil {
			included[minIdx].done = true
		} else {
			included[minIdx].ci = ci
			included[minIdx].data = data
		}
	}

	return buf.Bytes(), nil
}
```

**Step 4: Update mergePcaps to delegate**

Replace the existing `mergePcaps` function body with:

```go
func mergePcaps(pcap1, pcap2 []byte) ([]byte, error) {
	return mergeMultiplePcaps([]io.Reader{
		bytes.NewReader(pcap1),
		bytes.NewReader(pcap2),
	})
}
```

Remove the old `packetEntry` struct (no longer needed) and the `"sort"` import if unused after this change.

**Step 5: Run all tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline/bin-timeline-manager
go test ./pkg/siphandler/... -v
```

Expected: ALL PASS. The existing `TestMergePcaps` tests should still pass since `mergePcaps` delegates to the new function.

**Important:** The existing `TestMergePcaps_LinkTypeMismatch` test expects an error from `mergePcaps` on link type mismatch. With the new `mergeMultiplePcaps`, mismatched sources are **excluded** (not errored). Since `mergePcaps` delegates to `mergeMultiplePcaps`, and only 2 sources are passed where one mismatches, the result will have only packets from the matching source — not an error. **You must update the `TestMergePcaps_LinkTypeMismatch` test** to expect success with only the first source's packets (or empty if both sources have no data packets, which they don't in the test since they're header-only).

Update `TestMergePcaps_LinkTypeMismatch`:
```go
func TestMergePcaps_LinkTypeMismatch(t *testing.T) {
	createPcapWithLinkType := func(linkType layers.LinkType) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, linkType)
		return buf.Bytes()
	}

	pcap1 := createPcapWithLinkType(layers.LinkTypeEthernet)
	pcap2 := createPcapWithLinkType(layers.LinkTypeRaw)

	// With mergeMultiplePcaps, mismatched sources are excluded (not errored).
	// Both sources are header-only (no packets), so result is a valid empty pcap.
	result, err := mergePcaps(pcap1, pcap2)
	if err != nil {
		t.Fatalf("expected no error for link type mismatch (excluded), got: %v", err)
	}

	reader, err := pcapgo.NewReader(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}

	count := 0
	for {
		_, _, err := reader.ReadPacketData()
		if err != nil {
			break
		}
		count++
	}
	if count != 0 {
		t.Errorf("expected 0 packets, got %d", count)
	}
}
```

**Step 6: Run all siphandler tests again**

```bash
go test ./pkg/siphandler/... -v
```

Expected: ALL PASS

**Step 7: Commit**

```bash
git add pkg/siphandler/main.go pkg/siphandler/main_test.go
git commit -m "NOJIRA-Merge-rtp-pcap-timeline

- bin-timeline-manager: Add mergeMultiplePcaps with k-way merge and delegate mergePcaps to it"
```

---

### Task 5: Extend sipHandler with GCS support and update GetPcap

**Files:**
- Modify: `bin-timeline-manager/pkg/siphandler/main.go`
- Modify: `bin-timeline-manager/cmd/timeline-manager/main.go`

**Step 1: Update sipHandler struct and constructor**

In `pkg/siphandler/main.go`, update:

```go
type sipHandler struct {
	homerHandler homerhandler.HomerHandler
	gcsReader    GCSReader
	gcsBucket    string
}

// NewSIPHandler creates a new SIPHandler.
// gcsReader and gcsBucket are optional — if gcsBucket is empty, GCS integration is disabled.
func NewSIPHandler(homerHandler homerhandler.HomerHandler, gcsReader GCSReader, gcsBucket string) SIPHandler {
	return &sipHandler{
		homerHandler: homerHandler,
		gcsReader:    gcsReader,
		gcsBucket:    gcsBucket,
	}
}
```

**Step 2: Add fetchRTPPcaps helper**

Add to `pkg/siphandler/main.go`. Add `"os"`, `"sync"`, and `"path/filepath"` to imports:

```go
const (
	gcsRTPPrefix  = "rtp-recordings/"
	gcsTimeout    = 30 * time.Second
)

// fetchRTPPcaps downloads RTP pcap files from GCS for the given SIP Call-ID.
// Returns open file handles and a cleanup function. Caller must call cleanup after use.
func (h *sipHandler) fetchRTPPcaps(ctx context.Context, sipCallID string) ([]*os.File, func(), error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "fetchRTPPcaps",
		"sip_callid": sipCallID,
	})

	if h.gcsReader == nil || h.gcsBucket == "" {
		return nil, func() {}, nil
	}

	gcsCtx, cancel := context.WithTimeout(ctx, gcsTimeout)
	defer cancel()

	prefix := gcsRTPPrefix + sipCallID + "-"
	objects, err := h.gcsReader.ListObjects(gcsCtx, prefix)
	if err != nil {
		log.Warnf("Could not list RTP pcap objects from GCS, continuing without RTP: %v", err)
		return nil, func() {}, nil
	}

	if len(objects) == 0 {
		log.Debug("No RTP pcap files found in GCS.")
		return nil, func() {}, nil
	}

	log.WithField("count", len(objects)).Debugf("Found RTP pcap files in GCS. sip_callid: %s", sipCallID)

	// Download all objects in parallel to temp files
	type downloadResult struct {
		file *os.File
		err  error
	}

	results := make([]downloadResult, len(objects))
	var wg sync.WaitGroup

	for i, objName := range objects {
		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()

			tmpFile, err := os.CreateTemp("", "rtp-pcap-*.pcap")
			if err != nil {
				results[idx] = downloadResult{err: fmt.Errorf("could not create temp file: %w", err)}
				return
			}

			if err := h.gcsReader.DownloadObject(gcsCtx, name, tmpFile); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				results[idx] = downloadResult{err: fmt.Errorf("could not download %s: %w", name, err)}
				return
			}

			// Seek back to start for reading
			if _, err := tmpFile.Seek(0, 0); err != nil {
				tmpFile.Close()
				os.Remove(tmpFile.Name())
				results[idx] = downloadResult{err: fmt.Errorf("could not seek temp file: %w", err)}
				return
			}

			log.WithField("object", filepath.Base(name)).Debugf("Downloaded RTP pcap file. object: %s", name)
			results[idx] = downloadResult{file: tmpFile}
		}(i, objName)
	}

	wg.Wait()

	// Collect successful downloads
	var files []*os.File
	var cleanupPaths []string

	for i, r := range results {
		if r.err != nil {
			log.WithField("object", objects[i]).Warnf("Skipping RTP pcap download: %v", r.err)
			continue
		}
		files = append(files, r.file)
		cleanupPaths = append(cleanupPaths, r.file.Name())
	}

	cleanup := func() {
		for _, f := range files {
			f.Close()
		}
		for _, p := range cleanupPaths {
			os.Remove(p)
		}
	}

	return files, cleanup, nil
}
```

**Step 3: Update GetPcap to include RTP pcaps**

Replace the `GetPcap` method body in `pkg/siphandler/main.go`:

```go
func (h *sipHandler) GetPcap(ctx context.Context, sipCallID string, fromTime, toTime time.Time) ([]byte, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "GetPcap",
		"sip_callid": sipCallID,
	})
	log.WithFields(logrus.Fields{
		"from_time": fromTime,
		"to_time":   toTime,
	}).Info("SIPHandler called - fetching PCAP data")

	// Fetch SIP PCAP (hepid 1)
	sipPcapData, err := h.homerHandler.GetPcap(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get SIP PCAP data from Homer. err: %v", err)
		return nil, err
	}
	log.WithField("sip_pcap_size", len(sipPcapData)).Debug("Retrieved SIP PCAP data.")

	if len(sipPcapData) == 0 {
		log.Debug("No SIP PCAP data available.")
		return []byte{}, nil
	}

	// Fetch RTCP PCAP (hepid 5) - non-fatal if this fails
	rtcpPcapData, err := h.homerHandler.GetRTCPPcap(ctx, sipCallID, fromTime, toTime)
	if err != nil {
		log.Warnf("Could not get RTCP PCAP data from Homer, continuing with SIP only: %v", err)
	} else {
		log.WithField("rtcp_pcap_size", len(rtcpPcapData)).Debug("Retrieved RTCP PCAP data.")
	}

	// Fetch RTP pcap files from GCS (best-effort)
	rtpFiles, cleanup, rtpErr := h.fetchRTPPcaps(ctx, sipCallID)
	if rtpErr != nil {
		log.Warnf("Could not fetch RTP pcaps from GCS: %v", rtpErr)
	}
	defer cleanup()

	// Build merge sources
	var sources []io.Reader
	sources = append(sources, bytes.NewReader(sipPcapData))
	if len(rtcpPcapData) > 0 {
		sources = append(sources, bytes.NewReader(rtcpPcapData))
	}
	for _, f := range rtpFiles {
		sources = append(sources, f)
	}

	// Merge all sources
	var mergedData []byte
	if len(sources) > 1 {
		mergedData, err = mergeMultiplePcaps(sources)
		if err != nil {
			log.Warnf("Could not merge PCAPs, using SIP only: %v", err)
			mergedData = sipPcapData
		} else {
			log.WithFields(logrus.Fields{
				"merged_pcap_size": len(mergedData),
				"source_count":     len(sources),
			}).Debug("Merged all PCAP sources.")
		}
	} else {
		mergedData = sipPcapData
	}

	// Filter internal packets from PCAP
	filteredData, err := filterInternalPackets(mergedData)
	if err != nil {
		log.Warnf("Could not filter PCAP data, returning unfiltered: %v", err)
		return mergedData, nil
	}

	log.WithFields(logrus.Fields{
		"original_size": len(mergedData),
		"filtered_size": len(filteredData),
	}).Debug("Filtered internal packets from PCAP.")

	return filteredData, nil
}
```

**Step 4: Update main.go call site**

In `cmd/timeline-manager/main.go`, update `runServices()`:

```go
func runServices() error {
	db := dbhandler.NewHandler(config.Get().ClickHouseAddress, config.Get().ClickHouseDatabase)

	sockHandler := sockhandler.NewSockHandler(sock.TypeRabbitMQ, config.Get().RabbitMQAddress)
	sockHandler.Connect()

	evtHandler := eventhandler.NewEventHandler(db)

	homerH := homerhandler.NewHomerHandler(config.Get().HomerAPIAddress, config.Get().HomerAuthToken)

	// Initialize GCS reader for RTP pcap fetching (optional)
	var gcsReader siphandler.GCSReader
	gcsBucket := config.Get().GCSBucketName
	if gcsBucket != "" {
		client, err := storage.NewClient(context.Background())
		if err != nil {
			logrus.Warnf("Could not create GCS client, RTP pcap merge disabled: %v", err)
		} else {
			gcsReader = siphandler.NewGCSReader(client, gcsBucket)
			logrus.WithField("bucket", gcsBucket).Info("GCS reader initialized for RTP pcap merge.")
		}
	}

	sipH := siphandler.NewSIPHandler(homerH, gcsReader, gcsBucket)

	if errListen := runListen(sockHandler, evtHandler, sipH); errListen != nil {
		return errors.Wrapf(errListen, "failed to run service listen")
	}

	return nil
}
```

Add imports to `cmd/timeline-manager/main.go`:
```go
"context"

"cloud.google.com/go/storage"
```

**Step 5: Update all test NewSIPHandler calls**

All existing tests that call `NewSIPHandler(mockHomer)` must be updated to `NewSIPHandler(mockHomer, nil, "")`. Search for `NewSIPHandler` in `main_test.go` and update every occurrence. The `nil` GCSReader and empty bucket disable GCS integration in tests.

Affected lines (by current test function):
- `TestNewSIPHandler`: `NewSIPHandler(mockHomer)` → `NewSIPHandler(mockHomer, nil, "")`
- `TestGetSIPAnalysis`: `NewSIPHandler(mockHomer)` → `NewSIPHandler(mockHomer, nil, "")`
- `TestGetPcap`: `NewSIPHandler(mockHomer)` → `NewSIPHandler(mockHomer, nil, "")`
- `TestGetPcap_EmptyRTCP`: `NewSIPHandler(mockHomer)` → `NewSIPHandler(mockHomer, nil, "")`
- `TestGetPcap_MergeError`: `NewSIPHandler(mockHomer)` → `NewSIPHandler(mockHomer, nil, "")`
- `TestGetPcap_FilterError`: `NewSIPHandler(mockHomer)` → `NewSIPHandler(mockHomer, nil, "")`
- `TestGetPcap_MergeSuccess`: `NewSIPHandler(mockHomer)` → `NewSIPHandler(mockHomer, nil, "")`

**Step 6: Regenerate mocks and run tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline/bin-timeline-manager
go generate ./...
go test ./... -v
```

Expected: ALL PASS

**Step 7: Commit**

```bash
git add pkg/siphandler/main.go pkg/siphandler/main_test.go pkg/siphandler/mock_main.go cmd/timeline-manager/main.go
git commit -m "NOJIRA-Merge-rtp-pcap-timeline

- bin-timeline-manager: Extend GetPcap to fetch and merge RTP pcaps from GCS
- bin-timeline-manager: Update NewSIPHandler to accept GCSReader and bucket name
- bin-timeline-manager: Initialize GCS client in main.go when GCS_BUCKET_NAME is set"
```

---

### Task 6: Write integration tests for GetPcap with mock GCS

**Files:**
- Modify: `bin-timeline-manager/pkg/siphandler/main_test.go`

**Step 1: Add tests for GetPcap with GCS integration**

```go
func TestGetPcap_WithGCSRTPPcaps(t *testing.T) {
	fromTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	toTime := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	createPcapWithTimestamp := func(srcIP, dstIP string, ts time.Time) []byte {
		var buf bytes.Buffer
		writer := pcapgo.NewWriter(&buf)
		_ = writer.WriteFileHeader(65536, layers.LinkTypeEthernet)

		eth := &layers.Ethernet{
			SrcMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			DstMAC:       []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			EthernetType: layers.EthernetTypeIPv4,
		}
		ip := &layers.IPv4{
			Version:  4,
			SrcIP:    []byte(srcIP),
			DstIP:    []byte(dstIP),
			Protocol: layers.IPProtocolUDP,
		}
		udp := &layers.UDP{SrcPort: 5060, DstPort: 5060}
		_ = udp.SetNetworkLayerForChecksum(ip)

		packetBuf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{ComputeChecksums: true}
		_ = gopacket.SerializeLayers(packetBuf, opts, eth, ip, udp, gopacket.Payload([]byte("test")))
		packetData := packetBuf.Bytes()

		ci := gopacket.CaptureInfo{Timestamp: ts, CaptureLength: len(packetData), Length: len(packetData)}
		_ = writer.WritePacket(ci, packetData)

		return buf.Bytes()
	}

	t.Run("GCS returns RTP pcaps that get merged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)
		mockGCS := NewMockGCSReader(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))
		rtpPcap := createPcapWithTimestamp("\xcb\x00\x71\x01", "\x0a\x00\x00\x01", fromTime.Add(2*time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)

		mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-1-").Return(
			[]string{"rtp-recordings/call-1-ssrc1.pcap"}, nil,
		)
		mockGCS.EXPECT().DownloadObject(gomock.Any(), "rtp-recordings/call-1-ssrc1.pcap", gomock.Any()).DoAndReturn(
			func(_ context.Context, _ string, dest io.Writer) error {
				_, err := dest.Write(rtpPcap)
				return err
			},
		)

		h := NewSIPHandler(mockHomer, mockGCS, "test-bucket")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reader, err := pcapgo.NewReader(bytes.NewReader(result))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}

		count := 0
		for {
			_, _, err := reader.ReadPacketData()
			if err != nil {
				break
			}
			count++
		}
		if count != 2 {
			t.Errorf("expected 2 packets (SIP + RTP), got %d", count)
		}
	})

	t.Run("GCS list error degrades gracefully", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)
		mockGCS := NewMockGCSReader(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)
		mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-1-").Return(nil, fmt.Errorf("GCS unavailable"))

		h := NewSIPHandler(mockHomer, mockGCS, "test-bucket")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected SIP-only result, got empty")
		}
	})

	t.Run("GCS empty list returns SIP only", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)
		mockGCS := NewMockGCSReader(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)
		mockGCS.EXPECT().ListObjects(gomock.Any(), "rtp-recordings/call-1-").Return([]string{}, nil)

		h := NewSIPHandler(mockHomer, mockGCS, "test-bucket")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected SIP-only result, got empty")
		}
	})

	t.Run("GCS disabled when bucket empty", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockHomer := homerhandler.NewMockHomerHandler(mc)

		sipPcap := createPcapWithTimestamp("\x0a\x00\x00\x01", "\xcb\x00\x71\x01", fromTime.Add(time.Second))

		mockHomer.EXPECT().GetPcap(gomock.Any(), "call-1", fromTime, toTime).Return(sipPcap, nil)
		mockHomer.EXPECT().GetRTCPPcap(gomock.Any(), "call-1", fromTime, toTime).Return([]byte{}, nil)
		// No GCS mock expectations — GCS should not be called

		h := NewSIPHandler(mockHomer, nil, "")

		result, err := h.GetPcap(context.Background(), "call-1", fromTime, toTime)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) == 0 {
			t.Error("expected SIP-only result, got empty")
		}
	})
}
```

Add `"io"` to the test file imports if not already present.

**Step 2: Run tests**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline/bin-timeline-manager
go test ./pkg/siphandler/... -v
```

Expected: ALL PASS

**Step 3: Commit**

```bash
git add pkg/siphandler/main_test.go
git commit -m "NOJIRA-Merge-rtp-pcap-timeline

- bin-timeline-manager: Add integration tests for GetPcap with mock GCS (merge, graceful degradation, disabled)"
```

---

### Task 7: Update k8s deployment

**Files:**
- Modify: `bin-timeline-manager/k8s/deployment.yml`

**Step 1: Add GCS_BUCKET_NAME env var**

Add after the `HOMER_AUTH_TOKEN` env entry:

```yaml
            - name: GCS_BUCKET_NAME
              value: ${GCS_BUCKET_NAME}
```

**Step 2: Increase memory limit**

Update the resources section. Increase memory limit from 30Mi to 64Mi to accommodate GCS download I/O and pcap merge buffers:

```yaml
          resources:
            requests:
              cpu: "3m"
              memory: "3Mi"
            limits:
              cpu: "30m"
              memory: "64Mi"
```

**Step 3: Commit**

```bash
git add k8s/deployment.yml
git commit -m "NOJIRA-Merge-rtp-pcap-timeline

- bin-timeline-manager: Add GCS_BUCKET_NAME env var to k8s deployment
- bin-timeline-manager: Increase memory limit to 64Mi for RTP pcap merge workload"
```

---

### Task 8: Run full verification workflow

**Step 1: Run complete verification**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline/bin-timeline-manager
go mod tidy && \
go mod vendor && \
go generate ./... && \
go test ./... && \
golangci-lint run -v --timeout 5m
```

Expected: ALL PASS, no lint errors

**Step 2: Fix any issues found**

If tests or linting fail, fix the issues and re-run. Common issues:
- Unused imports (remove `"sort"` if `mergePcaps` no longer uses it)
- Missing `"io"` import in main.go or test file
- Mock regeneration needed after interface changes

**Step 3: Commit any fixes**

If fixes were needed:
```bash
git add -A
git commit -m "NOJIRA-Merge-rtp-pcap-timeline

- bin-timeline-manager: Fix lint and test issues from verification workflow"
```

---

### Task 9: Final review and PR preparation

**Step 1: Check for conflicts with main**

```bash
cd ~/gitvoipbin/monorepo/.worktrees/NOJIRA-Merge-rtp-pcap-timeline
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"
git log --oneline HEAD..origin/main
```

Expected: No conflicts

**Step 2: Review all changes**

```bash
git log --oneline main..HEAD
git diff main..HEAD --stat
```

Verify the diff includes:
- `docs/plans/2026-03-10-merge-rtp-pcap-timeline-design.md` (design doc)
- `docs/plans/2026-03-10-merge-rtp-pcap-timeline-impl-plan.md` (this plan)
- `bin-timeline-manager/internal/config/main.go` (GCSBucketName config)
- `bin-timeline-manager/internal/config/main_test.go` (config tests)
- `bin-timeline-manager/pkg/siphandler/gcsreader.go` (interface)
- `bin-timeline-manager/pkg/siphandler/gcsreader_impl.go` (implementation)
- `bin-timeline-manager/pkg/siphandler/mock_gcsreader.go` (generated mock)
- `bin-timeline-manager/pkg/siphandler/main.go` (merge logic + GCS fetch)
- `bin-timeline-manager/pkg/siphandler/main_test.go` (all tests)
- `bin-timeline-manager/pkg/siphandler/mock_main.go` (regenerated)
- `bin-timeline-manager/cmd/timeline-manager/main.go` (GCS client init)
- `bin-timeline-manager/go.mod` / `go.sum` (GCS dependency)
- `bin-timeline-manager/k8s/deployment.yml` (env var + memory)

**Step 3: Push and create PR**

```bash
git push -u origin NOJIRA-Merge-rtp-pcap-timeline
```

Then create PR with title `NOJIRA-Merge-rtp-pcap-timeline`.

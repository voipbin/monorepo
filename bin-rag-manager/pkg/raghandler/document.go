package raghandler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	dbchunk "monorepo/bin-rag-manager/models/chunk"
	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/pkg/chunker"
)

const (
	maxFileSize       = 50 * 1024 * 1024 // 50 MB
	maxRetryCount     = 3
	maxTokensPerChunk = 512
	heartbeatInterval = 10 // Update heartbeat every 10 chunks
	urlDownloadTimeout = 5 * time.Minute
)

func (h *ragHandler) DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "DocumentGet",
		"id":   id,
	})

	d, err := h.dbHandler.DocumentGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get document. err: %v", err)
		return nil, fmt.Errorf("could not get document: %w", err)
	}
	log.WithField("document", d).Debugf("Retrieved document. document_id: %s", d.ID)

	return d, nil
}

func (h *ragHandler) DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":    "DocumentList",
		"size":    size,
		"token":   token,
		"filters": filters,
	})

	docs, err := h.dbHandler.DocumentList(ctx, size, token, filters)
	if err != nil {
		log.Errorf("Could not list documents. err: %v", err)
		return nil, fmt.Errorf("could not list documents: %w", err)
	}
	log.Debugf("Listed documents. count: %d", len(docs))

	return docs, nil
}

// documentCreateInternal creates a document internally (not exposed via API).
// Uses the request ctx for DB operations (these must complete before the goroutine launches).
func (h *ragHandler) documentCreateInternal(ctx context.Context, customerID, ragID uuid.UUID, storageFileID uuid.UUID, sourceURL string) (*document.Document, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "documentCreateInternal",
		"customer_id": customerID,
		"rag_id":      ragID,
	})
	id, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("could not create uuid: %w", err)
	}

	docType := document.DocTypeURL
	name := sourceURL
	if storageFileID != uuid.Nil {
		docType = document.DocTypeUploaded
		name = storageFileID.String()
	}

	d := &document.Document{
		ID:            id,
		CustomerID:    customerID,
		RagID:         ragID,
		Name:          name,
		DocType:       docType,
		StorageFileID: storageFileID,
		SourceURL:     sourceURL,
		Status:        document.StatusPending,
	}

	if err := h.dbHandler.DocumentCreate(ctx, d); err != nil {
		return nil, fmt.Errorf("could not create document: %w", err)
	}

	res, err := h.dbHandler.DocumentGet(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get created document: %w", err)
	}
	log.WithField("document", res).Debugf("Created document internally. document_id: %s", res.ID)

	return res, nil
}

// documentIngest is the core async ingestion pipeline.
// It runs in a goroutine with context.Background().
// Acquires the ingestion semaphore to limit concurrent processing.
func (h *ragHandler) documentIngest(doc *document.Document) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "documentIngest",
		"document_id": doc.ID,
	})

	// Acquire semaphore slot to limit concurrent ingestion
	h.ingestSem <- struct{}{}
	defer func() { <-h.ingestSem }()

	ctx := context.Background()

	// Step 1: Atomic claim
	claimed, err := h.dbHandler.DocumentClaimForProcessing(ctx, doc.ID)
	if err != nil {
		log.Debugf("Could not claim document for processing, likely already claimed. document_id: %s, err: %v", doc.ID, err)
		return
	}
	log.WithField("document", claimed).Debugf("Claimed document for processing. document_id: %s, retry_count: %d", claimed.ID, claimed.RetryCount)

	// Step 2: Acquire file
	tmpPath, filename, contentType, err := h.documentAcquireFile(ctx, claimed)
	if err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not acquire file: %w", err))
		return
	}
	defer func() { _ = os.Remove(tmpPath) }()
	log.Debugf("Acquired file. document_id: %s, filename: %s, tmpPath: %s", claimed.ID, filename, tmpPath)

	// Step 3: Detect format
	ext := chunker.DetectExtensionFromFilename(filename)
	if ext == "" {
		ext = chunker.DetectExtensionFromContentType(contentType)
	}
	c := chunker.GetChunkerByExtension(ext)
	log.Debugf("Detected format. document_id: %s, ext: %s", claimed.ID, ext)

	// Step 4: Chunk content
	chunks, err := c.Chunk(tmpPath, maxTokensPerChunk)
	if err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not chunk file: %w", err))
		return
	}
	if len(chunks) == 0 {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("file produced no chunks"))
		return
	}
	log.Debugf("Chunked file. document_id: %s, chunk_count: %d", claimed.ID, len(chunks))

	// Step 5: Update heartbeat after chunking
	if err := h.dbHandler.DocumentUpdateHeartbeat(ctx, claimed.ID); err != nil {
		log.Errorf("Could not update heartbeat after chunking. document_id: %s, err: %v", claimed.ID, err)
	}

	// Step 6: Generate embeddings
	texts := make([]string, len(chunks))
	for i, ch := range chunks {
		texts[i] = ch.Text
	}

	embeddings, err := h.embedTextsWithHeartbeat(ctx, claimed.ID, texts)
	if err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not generate embeddings: %w", err))
		return
	}
	log.Debugf("Generated embeddings. document_id: %s, embedding_count: %d", claimed.ID, len(embeddings))

	// Step 7: Store chunks
	dbChunks, err := h.buildDBChunks(claimed, chunks)
	if err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not build db chunks: %w", err))
		return
	}

	if err := h.dbHandler.ChunkCreateBatch(ctx, dbChunks, embeddings); err != nil {
		h.documentIngestFail(ctx, claimed, fmt.Errorf("could not store chunks: %w", err))
		return
	}
	log.Debugf("Stored chunks. document_id: %s, chunk_count: %d", claimed.ID, len(dbChunks))

	// Step 8: Set status to ready
	if err := h.dbHandler.DocumentUpdate(ctx, claimed.ID, map[document.Field]any{
		document.FieldStatus: document.StatusReady,
	}); err != nil {
		log.Errorf("Could not set document to ready. document_id: %s, err: %v", claimed.ID, err)
		return
	}
	log.Infof("Document ingestion complete. document_id: %s, chunks: %d", claimed.ID, len(dbChunks))
}

// documentIngestFail handles ingestion failure with retry logic.
func (h *ragHandler) documentIngestFail(ctx context.Context, doc *document.Document, ingestErr error) {
	log := logrus.WithField("func", "documentIngestFail")
	log.Errorf("Document ingestion failed. document_id: %s, retry_count: %d, err: %v", doc.ID, doc.RetryCount, ingestErr)

	if doc.RetryCount >= maxRetryCount {
		// Terminal failure
		_ = h.dbHandler.DocumentUpdate(ctx, doc.ID, map[document.Field]any{
			document.FieldStatus:        document.StatusError,
			document.FieldStatusMessage: ingestErr.Error(),
		})
		return
	}

	// Reset to pending for retry
	_ = h.dbHandler.DocumentUpdate(ctx, doc.ID, map[document.Field]any{
		document.FieldStatus: document.StatusPending,
	})
}

// documentAcquireFile downloads the file to a temp file.
// Returns tmpPath, filename, contentType.
func (h *ragHandler) documentAcquireFile(ctx context.Context, doc *document.Document) (string, string, string, error) {
	if doc.StorageFileID != uuid.Nil {
		return h.documentDownloadGCS(ctx, doc.StorageFileID)
	}
	if doc.SourceURL != "" {
		return h.documentDownloadURL(ctx, doc.SourceURL)
	}
	return "", "", "", fmt.Errorf("document has no storage_file_id or source_url")
}

// documentDownloadGCS fetches file metadata from storage-manager and downloads from GCS.
func (h *ragHandler) documentDownloadGCS(ctx context.Context, storageFileID uuid.UUID) (string, string, string, error) {
	log := logrus.WithField("func", "documentDownloadGCS")
	file, err := h.reqHandler.StorageV1FileGet(ctx, storageFileID)
	if err != nil {
		return "", "", "", fmt.Errorf("could not get file metadata: %w", err)
	}
	log.WithField("file", file).Debugf("Retrieved storage file metadata. file_id: %s", storageFileID)

	if file.Filesize > maxFileSize {
		return "", "", "", fmt.Errorf("file size %d exceeds limit %d", file.Filesize, maxFileSize)
	}

	bucketName := file.BucketName
	if bucketName == "" {
		bucketName = h.bucketName
	}

	tmpPath, err := h.bucketReader.DownloadToTempFile(ctx, bucketName, file.Filepath)
	if err != nil {
		return "", "", "", fmt.Errorf("could not download from GCS: %w", err)
	}

	return tmpPath, file.Filename, "", nil
}

// documentDownloadURL downloads a URL to a temp file with size limit enforcement.
func (h *ragHandler) documentDownloadURL(ctx context.Context, sourceURL string) (string, string, string, error) {
	// Validate URL scheme to prevent SSRF (e.g., file://, ftp://)
	parsedSourceURL, err := url.Parse(sourceURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsedSourceURL.Scheme != "http" && parsedSourceURL.Scheme != "https" {
		return "", "", "", fmt.Errorf("unsupported URL scheme %q: only http and https are allowed", parsedSourceURL.Scheme)
	}

	dlCtx, cancel := context.WithTimeout(ctx, urlDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", "", "", fmt.Errorf("could not create request: %w", err)
	}

	httpClient := &http.Client{Timeout: urlDownloadTimeout}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", "", fmt.Errorf("could not fetch URL: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("URL returned status %d", resp.StatusCode)
	}

	if resp.ContentLength > maxFileSize {
		return "", "", "", fmt.Errorf("content length %d exceeds limit %d", resp.ContentLength, maxFileSize)
	}

	tmpFile, err := os.CreateTemp("", "rag_url_*")
	if err != nil {
		return "", "", "", fmt.Errorf("could not create temp file: %w", err)
	}

	limitReader := io.LimitReader(resp.Body, maxFileSize+1)
	n, err := io.Copy(tmpFile, limitReader)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", "", "", fmt.Errorf("could not download URL: %w", err)
	}
	if n > maxFileSize {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", "", "", fmt.Errorf("download size %d exceeds limit %d", n, maxFileSize)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", "", "", fmt.Errorf("could not close temp file: %w", err)
	}

	// Extract filename from URL path
	parsedURL, _ := url.Parse(sourceURL)
	filename := ""
	if parsedURL != nil {
		filename = path.Base(parsedURL.Path)
	}

	contentType := resp.Header.Get("Content-Type")
	return tmpFile.Name(), filename, contentType, nil
}

// embedTextsWithHeartbeat generates embeddings in batches and updates heartbeat periodically.
// Uses EmbedTexts (RETRIEVAL_DOCUMENT task type) — correct for indexing document chunks.
// EmbedText uses RETRIEVAL_QUERY task type and is for query-time only.
func (h *ragHandler) embedTextsWithHeartbeat(ctx context.Context, docID uuid.UUID, texts []string) ([][]float32, error) {
	log := logrus.WithField("func", "embedTextsWithHeartbeat")
	allEmbeddings := make([][]float32, 0, len(texts))

	for i := 0; i < len(texts); i += heartbeatInterval {
		end := i + heartbeatInterval
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embs, err := h.embedder.EmbedTexts(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("could not embed chunks %d-%d: %w", i, end-1, err)
		}
		allEmbeddings = append(allEmbeddings, embs...)

		if err := h.dbHandler.DocumentUpdateHeartbeat(ctx, docID); err != nil {
			log.Errorf("Could not update heartbeat during embedding. document_id: %s, err: %v", docID, err)
		}
	}

	return allEmbeddings, nil
}

// buildDBChunks converts chunker Chunks to database Chunk models.
func (h *ragHandler) buildDBChunks(doc *document.Document, chunks []chunker.Chunk) ([]*dbchunk.Chunk, error) {
	dbChunks := make([]*dbchunk.Chunk, 0, len(chunks))

	for i, ch := range chunks {
		id, err := uuid.NewV4()
		if err != nil {
			return nil, fmt.Errorf("could not create uuid: %w", err)
		}
		dbChunks = append(dbChunks, &dbchunk.Chunk{
			ID:           id,
			DocumentID:   doc.ID,
			RagID:        doc.RagID,
			CustomerID:   doc.CustomerID,
			ChunkIndex:   i,
			Text:         ch.Text,
			SectionTitle: ch.SectionTitle,
			TokenCount:   len(ch.Text) / 4,
		})
	}

	return dbChunks, nil
}

// DocumentIngestPendingAll resets stale processing documents and triggers pending ones.
// Called on startup before accepting requests.
// Uses 2-minute threshold so documents actively being processed by other pods are not reset.
func (h *ragHandler) DocumentIngestPendingAll(ctx context.Context) {
	log := logrus.WithField("func", "DocumentIngestPendingAll")
	// Reset stale processing documents (2min threshold avoids resetting docs actively processed by other pods)
	if err := h.dbHandler.DocumentResetStaleToPending(ctx, 2*time.Minute); err != nil {
		log.Errorf("Could not reset processing documents on startup: %v", err)
	}

	// Trigger pending documents
	docs, err := h.dbHandler.DocumentGetPending(ctx)
	if err != nil {
		log.Errorf("Could not get pending documents on startup: %v", err)
		return
	}

	log.Infof("Startup sweep: found %d pending documents to process", len(docs))
	for _, doc := range docs {
		go h.documentIngest(doc)
	}
}

// RunIngestionTicker periodically checks for stale and pending documents.
func (h *ragHandler) RunIngestionTicker(ctx context.Context, interval time.Duration) {
	log := logrus.WithField("func", "RunIngestionTicker")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Infof("Ingestion ticker stopped")
			return
		case <-ticker.C:
			h.ingestionTickerRun(ctx, interval)
		}
	}
}

func (h *ragHandler) ingestionTickerRun(ctx context.Context, threshold time.Duration) {
	log := logrus.WithField("func", "ingestionTickerRun")
	// Reset stale processing documents
	if err := h.dbHandler.DocumentResetStaleToPending(ctx, threshold); err != nil {
		log.Errorf("Ticker: could not reset stale documents: %v", err)
	}

	// Trigger pending documents
	docs, err := h.dbHandler.DocumentGetPending(ctx)
	if err != nil {
		log.Errorf("Ticker: could not get pending documents: %v", err)
		return
	}

	if len(docs) > 0 {
		log.Infof("Ticker: found %d pending documents to process", len(docs))
	}
	for _, doc := range docs {
		go h.documentIngest(doc)
	}
}

package messagehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	smfile "monorepo/bin-storage-manager/models/file"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/telnyx"
)

const (
	// inboundMediaMaxIngest is the number of ingests one process runs at the same time. The media of a message
	// is held in memory, so this bounds the memory. An ingest that finds no free slot is dropped (result busy).
	inboundMediaMaxIngest = 2

	// inboundMediaIngestTimeout is the deadline of one ingest: all downloads and all storage calls together.
	inboundMediaIngestTimeout = 90 * time.Second

	// inboundMediaStorageTimeout is the timeout of one storage call, in milliseconds. It is limited by the
	// deadline of the ingest as well.
	inboundMediaStorageTimeout = 30000
)

// startInboundMediaIngest starts the ingest of the media of a received message in the background, unless the
// process already runs the maximum number of ingests. It never blocks.
func (h *messageHandler) startInboundMediaIngest(ctx context.Context, m *message.Message, medias []telnyx.PayloadMedia) {
	select {
	case h.ingestSlots <- struct{}{}:
	default:
		logrus.WithField("message_id", m.ID).Warnf("Too many media ingests are running. The media of the message is dropped.")
		promInboundMediaTotal.WithLabelValues(string(FetchResultBusy)).Inc()
		return
	}

	// the request context ends with the answer of the hook.
	ingestCtx := context.WithoutCancel(ctx)
	go func() {
		defer func() { <-h.ingestSlots }()
		h.ingestInboundMedias(ingestCtx, m, medias)
	}()
}

// ingestInboundMedias downloads the media of a received message, stores each as a storage file and writes the
// list of the stored file ids to the message. A media that fails is counted and skipped; the others are kept.
func (h *messageHandler) ingestInboundMedias(ctx context.Context, m *message.Message, medias []telnyx.PayloadMedia) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "ingestInboundMedias",
		"message_id": m.ID,
		"count":      len(medias),
	})

	ctx, cancel := context.WithTimeout(ctx, inboundMediaIngestTimeout)
	defer cancel()

	count := func(result FetchResult, n int) {
		promInboundMediaTotal.WithLabelValues(string(result)).Add(float64(n))
	}

	if len(medias) > inboundMediaMaxCount {
		count(FetchResultOverflow, len(medias)-inboundMediaMaxCount)
		medias = medias[:inboundMediaMaxCount]
	}

	ids := []string{}
	budget := inboundMediaMaxTotalSize
	for i, media := range medias {
		if ctx.Err() != nil {
			count(FetchResultTimeout, len(medias)-i)
			break
		}

		fetched, result := h.mediaFetcher.Fetch(ctx, media, i, budget)
		if result != FetchResultOK || fetched == nil {
			count(result, 1)
			continue
		}

		f, err := h.reqHandler.StorageV1FileCreateContent(
			ctx,
			m.CustomerID,
			uuid.Nil,
			smfile.TypeMedia,
			fmt.Sprintf("media-%d", i),
			fmt.Sprintf("received with the message %s", m.ID),
			fetched.Filename,
			fetched.ContentType,
			fetched.Content,
			inboundMediaStorageTimeout,
		)
		if err != nil {
			log.Errorf("Could not store the media. index: %d, err: %v", i, err)
			count(FetchResultStorage, 1)
			continue
		}

		budget -= int64(len(fetched.Content))
		ids = append(ids, f.ID.String())
		count(FetchResultStored, 1)
	}

	if len(ids) == 0 {
		return
	}

	// the deadline of the ingest must not stop the write of what has been stored.
	writeCtx, writeCancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer writeCancel()

	if _, err := h.dbUpdateMedias(writeCtx, m.ID, ids); err != nil {
		log.Errorf("Could not update the medias of the message. err: %v", err)
	}
}

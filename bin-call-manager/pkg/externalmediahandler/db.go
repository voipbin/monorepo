package externalmediahandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"monorepo/bin-call-manager/models/externalmedia"
)

// Create creates a new external media
func (h *externalMediaHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	asteriskID string,
	channelID string,
	bridgeID string,
	playbackID string,
	referenceType externalmedia.ReferenceType,
	referenceID uuid.UUID,
	localIP string,
	localPort int,
	externalHost string,
	encapsulation externalmedia.Encapsulation,
	transport externalmedia.Transport,
	connectionType string,
	format string,
	directionListen externalmedia.Direction,
	directionSpeak externalmedia.Direction,
) (*externalmedia.ExternalMedia, error) {
	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
	}
	extMedia := &externalmedia.ExternalMedia{
		ID: id,

		AsteriskID: asteriskID,
		ChannelID:  channelID,
		BridgeID:   bridgeID,
		PlaybackID: playbackID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		Status: externalmedia.StatusRunning,

		LocalIP:         localIP,
		LocalPort:       localPort,
		ExternalHost:    externalHost,
		Encapsulation:   encapsulation,
		Transport:       transport,
		ConnectionType:  connectionType,
		Format:          format,
		DirectionListen: directionListen,
		DirectionSpeak:  directionSpeak,
	}

	if errDB := h.db.ExternalMediaSet(ctx, extMedia); errDB != nil {
		return nil, errors.Wrapf(errDB, "could not create the external media info to the database")
	}

	return extMedia, nil
}

// Get returns external media info
func (h *externalMediaHandler) Get(ctx context.Context, id uuid.UUID) (*externalmedia.ExternalMedia, error) {
	res, err := h.db.ExternalMediaGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get external media with id: %s", id)
	}

	return res, nil
}

// Gets returns list of external medias of the given filters.
func (h *externalMediaHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*externalmedia.ExternalMedia, error) {

	res := []*externalmedia.ExternalMedia{}
	if filters["reference_id"] != "" {
		referenceID := uuid.FromStringOrNil(filters["reference_id"])
		tmp, err := h.db.ExternalMediaGetByReferenceID(ctx, referenceID)
		if err != nil {
			// not found
			return res, nil
		}

		res = append(res, tmp)
	}

	return res, nil
}

// Gets returns list of external medias of the given filters.
func (h *externalMediaHandler) UpdateLocalAddress(ctx context.Context, id uuid.UUID, localIP string, localPort int) (*externalmedia.ExternalMedia, error) {
	tmp, err := h.db.ExternalMediaGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get external media with id: %s", id)
	}

	tmp.LocalIP = localIP
	tmp.LocalPort = localPort

	if errDB := h.db.ExternalMediaSet(ctx, tmp); errDB != nil {
		return nil, errors.Wrapf(errDB, "could not update external media local address")
	}

	res, err := h.db.ExternalMediaGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get external media with id: %s after set", id)
	}

	return res, nil
}

func (h *externalMediaHandler) UpdateStatus(ctx context.Context, id uuid.UUID, status externalmedia.Status) (*externalmedia.ExternalMedia, error) {
	tmp, err := h.db.ExternalMediaGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get external media with id: %s", id)
	}

	tmp.Status = status
	if errDB := h.db.ExternalMediaSet(ctx, tmp); errDB != nil {
		return nil, errors.Wrapf(errDB, "could not update external media status")
	}

	res, err := h.db.ExternalMediaGet(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get external media with id: %s after set", id)
	}

	return res, nil
}

package externalmediahandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/externalmedia"
)

// Create creates a new external media
func (h *externalMediaHandler) Create(
	ctx context.Context,
	id uuid.UUID,
	asteriskID string,
	channelID string,
	referenceType externalmedia.ReferenceType,
	referenceID uuid.UUID,
	localIP string,
	localPort int,
	externalHost string,
	encapsulation externalmedia.Encapsulation,
	transport externalmedia.Transport,
	connectionType string,
	format string,
	direction string,
) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":           "Create",
		"reference_type": referenceType,
		"reference_id":   referenceID,
	})

	if id == uuid.Nil {
		id = h.utilHandler.UUIDCreate()
	}
	extMedia := &externalmedia.ExternalMedia{
		ID: id,

		AsteriskID: asteriskID,
		ChannelID:  channelID,

		ReferenceType: referenceType,
		ReferenceID:   referenceID,

		LocalIP:        localIP,
		LocalPort:      localPort,
		ExternalHost:   externalHost,
		Encapsulation:  encapsulation,
		Transport:      transport,
		ConnectionType: connectionType,
		Format:         format,
		Direction:      direction,
	}

	if errDB := h.db.ExternalMediaSet(ctx, extMedia); errDB != nil {
		log.Errorf("Could not create the external media info to the database. err: %v", errDB)
		return nil, errDB
	}

	return extMedia, nil
}

// Get returns external media info
func (h *externalMediaHandler) Get(ctx context.Context, id uuid.UUID) (*externalmedia.ExternalMedia, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":              "Get",
		"external_media_id": id,
	})

	res, err := h.db.ExternalMediaGet(ctx, id)
	if err != nil {
		log.Errorf("Could not get external media. err: %v", err)
		return nil, err
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
		return nil, err
	}

	tmp.LocalIP = localIP
	tmp.LocalPort = localPort

	if errDB := h.db.ExternalMediaSet(ctx, tmp); errDB != nil {
		return nil, errDB
	}

	res, err := h.db.ExternalMediaGet(ctx, id)
	if err != nil {
		return nil, err
	}

	return res, nil
}

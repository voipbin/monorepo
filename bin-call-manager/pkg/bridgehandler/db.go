package bridgehandler

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-call-manager/models/bridge"
)

// Create creates a new bridge.
func (h *bridgeHandler) Create(
	ctx context.Context,

	asteriskID string,
	id string,
	name string,

	// info
	bridgeType bridge.Type,
	tech bridge.Tech,
	class string,
	creator string,

	videoMode string,
	videoSourceID string,

	// reference
	referenceType bridge.ReferenceType,
	referenceID uuid.UUID,
) (*bridge.Bridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Create",
		"bridge_id": id,
	})

	br := &bridge.Bridge{
		AsteriskID:    asteriskID,
		ID:            id,
		Name:          name,
		Type:          bridgeType,
		Tech:          tech,
		Class:         class,
		Creator:       creator,
		VideoMode:     videoMode,
		VideoSourceID: videoSourceID,
		ChannelIDs:    []string{},
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
	}

	if errCreate := h.db.BridgeCreate(ctx, br); errCreate != nil {
		log.Errorf("Could not create a bridge. err: %v", errCreate)
		return nil, errCreate
	}

	res, err := h.get(ctx, id)
	if err != nil {
		log.Errorf("Could not get created bridge. err: %v", err)
		return nil, err
	}
	promBridgeCreateTotal.WithLabelValues(string(referenceType)).Inc()

	return res, nil
}

// Get returns a bridge.
func (h *bridgeHandler) Get(ctx context.Context, id string) (*bridge.Bridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Get",
		"bridge_id": id,
	})

	// res, err := h.getWithTimeout(ctx, id, defaultExistTimeout)
	res, err := h.getWithTimeout(ctx, id, time.Second*10)
	if err != nil {
		log.Errorf("Could not get bridge. err: %v", err)
		return nil, err
	}

	return res, nil
}

// get returns a bridge where it gets from the database directly.
func (h *bridgeHandler) get(ctx context.Context, id string) (*bridge.Bridge, error) {
	res, err := h.db.BridgeGet(ctx, id)
	if err != nil {
		// we don't write log here
		// log.Errorf("Could not create a bridge. err: %v", err)
		return nil, errors.Wrapf(err, "could not get bridge from the database. bridge_id: %s", id)
	}

	return res, nil
}

// Delete deletes the channel.
func (h *bridgeHandler) Delete(ctx context.Context, id string) (*bridge.Bridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Delete",
		"bridge_id": id,
	})

	if errEnd := h.db.BridgeEnd(ctx, id); errEnd != nil {
		log.Errorf("Could not end the bridge. bridge_id: %s, err: %v", id, errEnd)
		return nil, errEnd
	}

	res, err := h.get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted bridge. err: %v", err)
		return nil, err
	}
	promBridgeDestroyedTotal.WithLabelValues(string(res.ReferenceType)).Inc()

	return res, nil
}

// AddChannelID adds the given channel id to the bridge.
func (h *bridgeHandler) AddChannelID(ctx context.Context, id, channelID string) (*bridge.Bridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "AddChannelID",
		"bridge_id":  id,
		"channel_id": channelID,
	})

	// add bridge's channel id
	if errAdd := h.db.BridgeAddChannelID(ctx, id, channelID); errAdd != nil {
		log.Errorf("Could not add the channel from the bridge. err: %v", errAdd)
		return nil, errAdd
	}

	res, err := h.get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted bridge. err: %v", err)
		return nil, err
	}

	return res, nil
}

// RemoveChannelID removes the given channel id from the bridge.
func (h *bridgeHandler) RemoveChannelID(ctx context.Context, id, channelID string) (*bridge.Bridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "RemoveChannelID",
		"bridge_id":  id,
		"channel_id": channelID,
	})

	// add bridge's channel id
	if errAdd := h.db.BridgeRemoveChannelID(ctx, id, channelID); errAdd != nil {
		log.Errorf("Could not add the channel from the bridge. err: %v", errAdd)
		return nil, errAdd
	}

	res, err := h.get(ctx, id)
	if err != nil {
		log.Errorf("Could not get deleted bridge. err: %v", err)
		return nil, err
	}

	return res, nil
}

// getWithTimeout gets the bridge with for given timeout.
func (h *bridgeHandler) getWithTimeout(ctx context.Context, id string, timeout time.Duration) (*bridge.Bridge, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	chanRes := make(chan *bridge.Bridge)

	go func() {
		defer close(chanRes)
		for {

			select {
			case <-cctx.Done():
				return

			default:
				tmp, err := h.get(cctx, id)
				if err != nil {
					time.Sleep(defaultDelayTimeout)
					continue
				}
				chanRes <- tmp
				return
			}
		}
	}()

	select {
	case res := <-chanRes:
		return res, nil
	case <-cctx.Done():
		return nil, fmt.Errorf("could not get a bridge. GetWithTimeout. err: %v", cctx.Err())
	}
}

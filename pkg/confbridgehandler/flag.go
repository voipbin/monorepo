package confbridgehandler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/confbridge"
)

// flagExist returns true if the given flag exists in the flags.
func (h *confbridgeHandler) flagExist(ctx context.Context, flags []confbridge.Flag, flag confbridge.Flag) bool {
	for _, f := range flags {
		if f == flag {
			return true
		}
	}

	return false
}

// FlagAdd adds the given flag to the flags
func (h *confbridgeHandler) FlagAdd(ctx context.Context, id uuid.UUID, flag confbridge.Flag) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "FlagAdd",
		"confbridge_id": id,
		"flag":          flag,
	})

	cb, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge. err: %v", err)
		return nil, errors.Wrap(err, "could not get confbridge")
	}

	if h.flagExist(ctx, cb.Flags, flag) {
		// already existed. nothing to do.
		return cb, nil
	}

	tmp := append(cb.Flags, flag)

	// update
	res, err := h.UpdateFlags(ctx, id, tmp)
	if err != nil {
		log.Errorf("Could not update the flags. err: %v", err)
		return nil, errors.Wrap(err, "could notupdate the flags")
	}

	return res, nil
}

// FlagRemove removes the given flag from the flags
func (h *confbridgeHandler) FlagRemove(ctx context.Context, id uuid.UUID, flag confbridge.Flag) (*confbridge.Confbridge, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "FlagRemove",
		"confbridge_id": id,
		"flag":          flag,
	})

	cb, err := h.Get(ctx, id)
	if err != nil {
		log.Errorf("Could not get confbridge. err: %v", err)
		return nil, errors.Wrap(err, "could not get confbridge")
	}

	idx := -1
	for i, f := range cb.Flags {
		if f == flag {
			// found
			idx = i
			break
		}
	}

	if idx == -1 {
		// given flag does not exist nothing to do
		return cb, nil
	}

	tmp := append(cb.Flags[:idx], cb.Flags[idx+1:]...)

	// update
	res, err := h.UpdateFlags(ctx, id, tmp)
	if err != nil {
		log.Errorf("Could not update the flags. err: %v", err)
		return nil, errors.Wrap(err, "could notupdate the flags")
	}

	return res, nil
}

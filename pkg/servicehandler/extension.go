package servicehandler

import (
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/requesthandler/models/rmextension"
)

// ExtensionCreate is a service handler for flow creation.
func (h *serviceHandler) ExtensionCreate(u *user.User, e *extension.Extension) (*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":      u.ID,
		"domain_id": e.DomainID,
		"name":      e.Name,
		"detail":    e.Detail,
	})
	log.Debug("Creating a new extension.")

	ext := rmextension.CreateDomain(e)
	ext.UserID = u.ID
	tmp, err := h.reqHandler.RMExtensionCreate(ext)
	if err != nil {
		log.Errorf("Could not create a new domain. err: %v", err)
		return nil, err
	}

	res := tmp.ConvertExtension()

	return res, nil
}

// ExtensionDelete deletes the extension of the given id.
func (h *serviceHandler) ExtensionDelete(u *user.User, id uuid.UUID) error {
	log := logrus.WithFields(logrus.Fields{
		"user":         u.ID,
		"username":     u.Username,
		"extension_id": id,
	})
	log.Debug("Deleting a extension.")

	// get extension
	ext, err := h.reqHandler.RMExtensionGet(id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return fmt.Errorf("could not find extension info. err: %v", err)
	}

	// permission check
	if u.HasPermission(user.PermissionAdmin) != true && ext.UserID != u.ID {
		log.Errorf("The user has no permission for this extension. user: %d, domain_user: %d", u.ID, ext.UserID)
		return fmt.Errorf("user has no permission")
	}

	if err := h.reqHandler.RMExtensionDelete(id); err != nil {
		return err
	}

	return nil
}

// ExtensionGet gets the extension of the given id.
// It returns extension if it succeed.
func (h *serviceHandler) ExtensionGet(u *user.User, id uuid.UUID) (*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":         u.ID,
		"username":     u.Username,
		"extension_id": id,
	})
	log.Debug("Getting a extension.")

	// get extension
	d, err := h.reqHandler.RMExtensionGet(id)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	// permission check
	if u.HasPermission(user.PermissionAdmin) != true && d.UserID != u.ID {
		log.Errorf("The user has no permission for this extension. user: %d, extension_user: %d", u.ID, d.UserID)
		return nil, fmt.Errorf("user has no permission")
	}

	res := d.ConvertExtension()
	return res, nil
}

// ExtensionGets gets the list of extensions of the given user id.
// It returns list of extensions if it succeed.
func (h *serviceHandler) ExtensionGets(u *user.User, domainID uuid.UUID, size uint64, token string) ([]*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":     u.ID,
		"username": u.Username,
		"size":     size,
		"token":    token,
	})
	log.Debug("Getting a extensions.")

	if token == "" {
		token = getCurTime()
	}

	// get extensions
	exts, err := h.reqHandler.RMExtensionGets(domainID, token, size)
	if err != nil {
		log.Errorf("Could not get extensions info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extensions info. err: %v", err)
	}

	// create result
	res := []*extension.Extension{}
	for _, ext := range exts {
		tmp := ext.ConvertExtension()
		res = append(res, tmp)
	}

	return res, nil
}

// ExtesnionUpdate updates the extension info.
// It returns updated extension if it succeed.
func (h *serviceHandler) ExtensionUpdate(u *user.User, d *extension.Extension) (*extension.Extension, error) {
	log := logrus.WithFields(logrus.Fields{
		"user":      u.ID,
		"username":  u.Username,
		"extension": d.ID,
	})
	log.Debug("Updating an extension.")

	// get extension
	tmpExtension, err := h.reqHandler.RMExtensionGet(d.ID)
	if err != nil {
		log.Errorf("Could not get extension info from the registrar-manager. err: %v", err)
		return nil, fmt.Errorf("could not find extension info. err: %v", err)
	}

	// check the ownership
	if u.Permission != user.PermissionAdmin && u.ID != tmpExtension.UserID {
		log.Info("The user has no permission for this extension.")
		return nil, fmt.Errorf("user has no permission")
	}

	reqExtension := rmextension.CreateDomain(d)
	res, err := h.reqHandler.RMExtensionUpdate(reqExtension)
	if err != nil {
		logrus.Errorf("Could not update the domain. err: %v", err)
		return nil, err
	}

	resExtension := res.ConvertExtension()
	return resExtension, nil
}

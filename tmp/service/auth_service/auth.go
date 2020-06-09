package auth_service

import "gitlab.com/voipbin/bin-manager/api-manager/models"

type Auth struct {
	Username string
	Password string
}

func (a *Auth) Check() (bool, error) {
	return models.CheckAuth(a.Username, a.Password)
}

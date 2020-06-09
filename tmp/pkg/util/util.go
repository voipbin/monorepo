package util

import "gitlab.com/voipbin/bin-manager/api-manager/pkg/setting"

// Setup Initialize the util
func Setup() {
	jwtSecret = []byte(setting.AppSetting.JwtSecret)
}

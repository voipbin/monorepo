package service

import (
	"regexp"

	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var validProvisioningToken = regexp.MustCompile(`^[0-9a-f]{64}$`)

// GetProvisioningExtension handles GET /provisioning/extension requests.
// It is an unauthenticated endpoint consumed by SIP softphones (Linphone
// remote provisioning): a valid short-lived token is exchanged for the
// extension's lpconfig XML. Every failure mode (bad token format, unknown or
// expired token, backend error) returns the same bare 400 with an empty body
// to resist token enumeration. The access log for this path is skipped
// (SkipPaths) so the token never reaches stdout; this handler emits a single
// structured log line (without the token) to preserve observability.
func GetProvisioningExtension(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":      "GetProvisioningExtension",
		"client_ip": c.ClientIP(),
	})
	if linphoneHeader := c.GetHeader("X-Linphone-Provisioning"); linphoneHeader != "" {
		log = log.WithField("x_linphone_provisioning", linphoneHeader)
	}

	token := c.Query("token")
	if !validProvisioningToken.MatchString(token) {
		log.WithField("result", "rejected").Info("Provisioning request rejected.")
		c.AbortWithStatus(400)
		return
	}

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	xmlBytes, err := serviceHandler.ExtensionProvisioningXMLGet(c.Request.Context(), token)
	if err != nil {
		// Keep the failure log generic: never log the token, and do not
		// distinguish unknown/expired/deleted so the log stream leaks
		// nothing an access log would not.
		log.WithField("result", "rejected").Info("Provisioning request rejected.")
		c.AbortWithStatus(400)
		return
	}

	log.WithField("result", "ok").Info("Provisioning request served.")
	c.Data(200, "application/xml; charset=utf-8", xmlBytes)
}

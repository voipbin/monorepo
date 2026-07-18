package server

import (
	"regexp"

	"monorepo/bin-api-manager/gens/openapi_server"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/sirupsen/logrus"
)

// regexHexColor matches the OpenAPI-declared theme_config hex color pattern
// (`^#[0-9a-fA-F]{6}$`). The OpenAPI spec only documents this constraint for
// client-side tooling/codegen; it is not enforced at runtime by the
// generated server code, so it must be checked explicitly here before the
// value is persisted and later rendered, unsanitized, into third-party
// pages via the embeddable widget.
var regexHexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

const (
	maxHeaderTitleLength    = 100
	maxHeaderSubtitleLength = 200
)

func (h *server) GetWebchatWidgets(c *gin.Context, params openapi_server.GetWebchatWidgetsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetWebchatWidgets",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	pageSize := uint64(100)
	if params.PageSize != nil {
		pageSize = uint64(*params.PageSize)
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	pageToken := ""
	if params.PageToken != nil {
		pageToken = *params.PageToken
	}

	tmps, err := h.serviceHandler.WebchatWidgetList(c.Request.Context(), a, pageSize, pageToken)
	if err != nil {
		logrus.Errorf("Could not get webchat widgets info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		if tmps[len(tmps)-1].TMCreate != nil {
			nextToken = tmps[len(tmps)-1].TMCreate.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}

	res := GenerateListResponse(tmps, nextToken)
	c.JSON(200, res)
}

func (h *server) PostWebchatWidgets(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostWebchatWidgets",
		"request_address": c.ClientIP,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	var req openapi_server.PostWebchatWidgetsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	sessionFlowID := uuid.FromStringOrNil(req.SessionFlowId)

	messageFlowID := uuid.Nil
	if req.MessageFlowId != nil {
		messageFlowID = uuid.FromStringOrNil(*req.MessageFlowId)
	}

	sessionIdleTimeout := 0
	if req.SessionIdleTimeout != nil {
		sessionIdleTimeout = *req.SessionIdleTimeout
	}

	themeConfig, themeErr := convertWebchatThemeConfig(req.ThemeConfig)
	if themeErr != nil {
		log.Infof("Invalid theme_config in request. err: %v", themeErr)
		abortWithError(c, themeErr)
		return
	}

	res, err := h.serviceHandler.WebchatWidgetCreate(
		c.Request.Context(),
		a,
		req.Name,
		sessionFlowID,
		messageFlowID,
		sessionIdleTimeout,
		themeConfig,
	)
	if err != nil {
		log.Errorf("Could not create a webchat widget. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteWebchatWidgetsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteWebchatWidgetsId",
		"request_address": c.ClientIP,
		"widget_id":       id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id.String())
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.WebchatWidgetDelete(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not delete the webchat widget. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetWebchatWidgetsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetWebchatWidgetsId",
		"request_address": c.ClientIP,
		"widget_id":       id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id.String())
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.WebchatWidgetGet(c.Request.Context(), a, target)
	if err != nil {
		log.Infof("Could not get the webchat widget info. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutWebchatWidgetsId(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutWebchatWidgetsId",
		"request_address": c.ClientIP,
		"widget_id":       id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id.String())
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	var req openapi_server.PutWebchatWidgetsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_JSON_BODY", "The request body is not valid JSON.").Wrap(err))
		return
	}

	sessionFlowID := uuid.FromStringOrNil(req.SessionFlowId)

	messageFlowID := uuid.Nil
	if req.MessageFlowId != nil {
		messageFlowID = uuid.FromStringOrNil(*req.MessageFlowId)
	}

	sessionIdleTimeout := 0
	if req.SessionIdleTimeout != nil {
		sessionIdleTimeout = *req.SessionIdleTimeout
	}

	themeConfig, themeErr := convertWebchatThemeConfig(req.ThemeConfig)
	if themeErr != nil {
		log.Infof("Invalid theme_config in request. err: %v", themeErr)
		abortWithError(c, themeErr)
		return
	}

	res, err := h.serviceHandler.WebchatWidgetUpdate(
		c.Request.Context(),
		a,
		target,
		req.Name,
		sessionFlowID,
		messageFlowID,
		sessionIdleTimeout,
		themeConfig,
	)
	if err != nil {
		log.Errorf("Could not update the webchat widget. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

func (h *server) PostWebchatWidgetsIdDirectHashRegenerate(c *gin.Context, id openapi_types.UUID) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostWebchatWidgetsIdDirectHashRegenerate",
		"request_address": c.ClientIP,
		"widget_id":       id,
	})

	a, ok := getAuthIdentity(c)
	if !ok {
		log.Errorf("Could not find auth identity.")
		abortWithError(c, cerrors.Unauthenticated(commonoutline.ServiceNameAPIManager, "AUTHENTICATION_REQUIRED", "Authentication is required."))
		return
	}
	log = log.WithField("agent", a)

	target := uuid.FromStringOrNil(id.String())
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		abortWithError(c, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_ID", "The provided id is not a valid UUID."))
		return
	}

	res, err := h.serviceHandler.WebchatWidgetDirectHashRegenerate(c.Request.Context(), a, target)
	if err != nil {
		log.Errorf("Could not regenerate webchat widget direct hash. err: %v", err)
		abortWithServiceError(c, err)
		return
	}

	c.JSON(200, res)
}

// convertWebchatThemeConfig converts the OpenAPI-generated theme_config
// request shape to the internal wcwidget.ThemeConfig, or nil when the
// request omitted it (all Widget theming fields stay optional end to end).
//
// Also enforces, at this handler boundary, the format constraints the
// OpenAPI spec documents (hex color pattern, theme_mode enum, header
// title/subtitle max lengths) but which the generated server code does
// NOT validate at runtime on its own. This matters because theme_config
// values are customer-controlled and get rendered, largely unsanitized,
// into the embeddable widget on third-party pages -- accepting arbitrary
// strings here would let a customer persist malformed or oversized values
// that downstream rendering code does not expect.
func convertWebchatThemeConfig(req *openapi_server.WebchatManagerWidgetThemeConfig) (*wcwidget.ThemeConfig, *cerrors.VoipbinError) {
	if req == nil {
		return nil, nil
	}

	res := &wcwidget.ThemeConfig{}
	if req.PrimaryColor != nil {
		if !regexHexColor.MatchString(*req.PrimaryColor) {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_THEME_CONFIG", "primary_color must match ^#[0-9a-fA-F]{6}$.")
		}
		res.PrimaryColor = *req.PrimaryColor
	}
	if req.SecondaryColor != nil {
		if !regexHexColor.MatchString(*req.SecondaryColor) {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_THEME_CONFIG", "secondary_color must match ^#[0-9a-fA-F]{6}$.")
		}
		res.SecondaryColor = *req.SecondaryColor
	}
	if req.HeaderBackgroundColor != nil {
		if !regexHexColor.MatchString(*req.HeaderBackgroundColor) {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_THEME_CONFIG", "header_background_color must match ^#[0-9a-fA-F]{6}$.")
		}
		res.HeaderBackgroundColor = *req.HeaderBackgroundColor
	}
	if req.HeaderTextColor != nil {
		if !regexHexColor.MatchString(*req.HeaderTextColor) {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_THEME_CONFIG", "header_text_color must match ^#[0-9a-fA-F]{6}$.")
		}
		res.HeaderTextColor = *req.HeaderTextColor
	}
	if req.LogoUrl != nil {
		res.LogoURL = *req.LogoUrl
	}
	if req.Position != nil {
		res.Position = wcwidget.WidgetPosition(*req.Position)
	}
	if req.ThemeMode != nil {
		mode := wcwidget.ThemeMode(*req.ThemeMode)
		switch mode {
		case wcwidget.ThemeModeLight, wcwidget.ThemeModeDark, wcwidget.ThemeModeAuto:
			res.ThemeMode = mode
		default:
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_THEME_CONFIG", "theme_mode must be one of light, dark, auto.")
		}
	}
	if req.HeaderTitle != nil {
		if len(*req.HeaderTitle) > maxHeaderTitleLength {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_THEME_CONFIG", "header_title must be at most 100 characters.")
		}
		res.HeaderTitle = *req.HeaderTitle
	}
	if req.HeaderSubtitle != nil {
		if len(*req.HeaderSubtitle) > maxHeaderSubtitleLength {
			return nil, cerrors.InvalidArgument(commonoutline.ServiceNameAPIManager, "INVALID_THEME_CONFIG", "header_subtitle must be at most 200 characters.")
		}
		res.HeaderSubtitle = *req.HeaderSubtitle
	}

	return res, nil
}

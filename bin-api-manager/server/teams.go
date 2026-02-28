package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	amteam "monorepo/bin-ai-manager/models/team"
	"monorepo/bin-api-manager/gens/openapi_server"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
)

func (h *server) PostTeams(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PostTeams",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	var req openapi_server.PostTeamsJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	startMemberID := uuid.FromStringOrNil(req.StartMemberId)
	members := convertOpenAPIMembers(req.Members)

	var parameter map[string]any
	if req.Parameter != nil {
		parameter = *req.Parameter
	}

	res, err := h.serviceHandler.TeamCreate(
		c.Request.Context(),
		&a,
		req.Name,
		req.Detail,
		startMemberID,
		members,
		parameter,
	)
	if err != nil {
		log.Errorf("Could not create a team. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) GetTeams(c *gin.Context, params openapi_server.GetTeamsParams) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTeams",
		"request_address": c.ClientIP,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

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

	tmps, err := h.serviceHandler.TeamGetsByCustomerID(c.Request.Context(), &a, pageSize, pageToken)
	if err != nil {
		log.Errorf("Could not get a team list. err: %v", err)
		c.AbortWithStatus(400)
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

func (h *server) GetTeamsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "GetTeamsId",
		"request_address": c.ClientIP,
		"team_id":         id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.TeamGet(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not get a team. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) DeleteTeamsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "DeleteTeamsId",
		"request_address": c.ClientIP,
		"team_id":         id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	res, err := h.serviceHandler.TeamDelete(c.Request.Context(), &a, target)
	if err != nil {
		log.Errorf("Could not delete the team. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

func (h *server) PutTeamsId(c *gin.Context, id string) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "PutTeamsId",
		"request_address": c.ClientIP,
		"team_id":         id,
	})

	tmp, exists := c.Get("agent")
	if !exists {
		log.Errorf("Could not find agent info.")
		c.AbortWithStatus(400)
		return
	}
	a := tmp.(amagent.Agent)
	log = log.WithFields(logrus.Fields{
		"agent": a,
	})

	target := uuid.FromStringOrNil(id)
	if target == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	var req openapi_server.PutTeamsIdJSONBody
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	startMemberID := uuid.FromStringOrNil(req.StartMemberId)
	members := convertOpenAPIMembers(req.Members)

	var parameter map[string]any
	if req.Parameter != nil {
		parameter = *req.Parameter
	}

	res, err := h.serviceHandler.TeamUpdate(
		c.Request.Context(),
		&a,
		target,
		req.Name,
		req.Detail,
		startMemberID,
		members,
		parameter,
	)
	if err != nil {
		log.Errorf("Could not update the team. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// convertOpenAPIMembers converts generated OpenAPI member structs to internal team.Member structs.
func convertOpenAPIMembers(apiMembers []openapi_server.AIManagerTeamMember) []amteam.Member {
	members := make([]amteam.Member, len(apiMembers))
	for i, m := range apiMembers {
		var transitions []amteam.Transition
		if m.Transitions != nil {
			transitions = make([]amteam.Transition, len(*m.Transitions))
			for j, t := range *m.Transitions {
				transitions[j] = amteam.Transition{
					FunctionName: t.FunctionName,
					Description:  t.Description,
					NextMemberID: uuid.FromStringOrNil(t.NextMemberId),
				}
			}
		}

		members[i] = amteam.Member{
			ID:          uuid.FromStringOrNil(m.Id),
			Name:        m.Name,
			AIID:        uuid.FromStringOrNil(m.AiId),
			Transitions: transitions,
		}
	}
	return members
}

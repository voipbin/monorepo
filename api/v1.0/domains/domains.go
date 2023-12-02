package domains

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// domainsPOST handles POST /domains request.
// It creates a new domain with the given info and returns created domain info.
// @Summary     Create a new domain and returns detail created domain info.
// @Description Create a new domain and returns detail created domain info.
// @Produce     json
// @Success     200 {object} domain.Domain
// @Router      /v1.0/domains [post]
func domainsPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "domainsPOST",
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

	var req request.BodyDomainsPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// create a domain
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	domain, err := serviceHandler.DomainCreate(c.Request.Context(), &a, req.DomainName, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not create a domain. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, domain)
}

// domainsPOST handles GET /domains request.
// It gets a list of domains with the given info.
// @Summary     Gets a list of domains.
// @Description Gets a list of domains
// @Produce     json
// @Param       page_size  query    int    false "The size of results. Max 100"
// @Param       page_token query    string false "The token. tm_create"
// @Success     200        {object} response.BodyDomainsGET
// @Router      /v1.0/domains [get]
func domainsGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "domainsGET",
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

	var req request.ParamDomainsGET
	if err := c.BindQuery(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}
	log.Debugf("domainsGET. Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get domains
	domains, err := serviceHandler.DomainGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		log.Errorf("Could not get a domain list. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(domains) > 0 {
		nextToken = domains[len(domains)-1].TMCreate
	}
	res := response.BodyDomainsGET{
		Result: domains,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// domainsIDGET handles GET /domains/{id} request.
// It returns detail domain info.
// @Summary     Returns detail domain info.
// @Description Returns detail domain info of the given domain id.
// @Produce     json
// @Param       id  path     string true "The ID of the domain"
// @Success     200 {object} domain.Domain
// @Router      /v1.0/domains/{id} [get]
func domainsIDGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "domainsIDGET",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("domain_id", id)
	log.Debug("Executing domainsIDGET.")

	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.DomainGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not get a domain. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// domainsIDPUT handles PUT /domains/{id} request.
// It updates a exist domain info with the given domain info.
// And returns updated domain info if it succeed.
// @Summary     Update a domain and reuturns updated domain info.
// @Description Update a domain and returns detail updated domain info.
// @Produce     json
// @Success     200 {object} domain.Domain
// @Router      /v1.0/domains/{id} [put]
func domainsIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "domainsIDPUT",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("domain_id", id)

	var req request.BodyDomainsIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update a domain
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.DomainUpdate(c.Request.Context(), &a, id, req.Name, req.Detail)
	if err != nil {
		log.Errorf("Could not update the domain. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// domainsIDDELETE handles DELETE /domains/{id} request.
// It deletes a exist domain info.
// @Summary     Delete a existing domain.
// @Description Delete a existing domain.
// @Produce     json
// @Success     200
// @Router      /v1.0/domains/{id} [delete]
func domainsIDDELETE(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "domainsIDDELETE",
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

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))
	log = log.WithField("domain_id", id)
	log.Debug("Executing domainsIDDELETE.")

	// delete a domain
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.DomainDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Errorf("Could not delete the domain. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

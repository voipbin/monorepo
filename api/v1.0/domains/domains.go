package domains

import (
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

// domainsPOST handles POST /domains request.
// It creates a new domain with the given info and returns created domain info.
// @Summary Create a new domain and returns detail created domain info.
// @Description Create a new domain and returns detail created domain info.
// @Produce json
// @Success 200 {object} domain.Domain
// @Router /v1.0/domains [post]
func domainsPOST(c *gin.Context) {

	var body request.BodyDomainsPOST
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(models.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// create a domain
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	domain, err := serviceHandler.DomainCreate(&u, body.DomainName, body.Name, body.Detail)
	if err != nil {
		log.Errorf("Could not create a domain. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, domain)
	return
}

// domainsPOST handles GET /domains request.
// It gets a list of domains with the given info.
// @Summary Gets a list of domains.
// @Description Gets a list of domains
// @Produce json
// @Success 200 {array} domain.Domain
// @Router /v1.0/domains [get]
func domainsGET(c *gin.Context) {

	var requestParam request.ParamDomainsGET

	if err := c.BindQuery(&requestParam); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log := logrus.WithFields(
		logrus.Fields{
			"request_address": c.ClientIP,
		},
	)
	log.Debugf("flowsGET. Received request detail. page_size: %d, page_token: %s", requestParam.PageSize, requestParam.PageToken)

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}

	u := tmp.(models.User)
	log = log.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// set max page size
	pageSize := requestParam.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	}

	// get service
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get domains
	domains, err := serviceHandler.DomainGets(&u, pageSize, requestParam.PageToken)
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
	return
}

// domainsIDGET handles GET /domains/{id} request.
// It returns detail domain info.
// @Summary Returns detail domain info.
// @Description Returns detail flow info of the given domain id.
// @Produce json
// @Param id path string true "The ID of the domain"
// @Param token query string true "JWT token"
// @Success 200 {object} domain.Domain
// @Router /v1.0/domains/{id} [get]
func domainsIDGET(c *gin.Context) {
	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(models.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})
	log.Debug("Executing domainsIDGET.")

	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.DomainGet(&u, id)
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
// @Summary Update a domain and reuturns updated domain info.
// @Description Update a domain and returns detail updated domain info.
// @Produce json
// @Success 200 {object} domain.Domain
// @Router /v1.0/domains/{id} [put]
func domainsIDPUT(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	var body request.BodyDomainsIDPUT
	if err := c.BindJSON(&body); err != nil {
		c.AbortWithStatus(400)
		return
	}

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(models.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	f := &models.Domain{
		ID:     id,
		Name:   body.Name,
		Detail: body.Detail,
	}

	// update a domain
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.DomainUpdate(&u, f)
	if err != nil {
		log.Errorf("Could not create a domain. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
	return
}

// domainsIDDELETE handles DELETE /domains/{id} request.
// It deletes a exist domain info.
// @Summary Delete a existing domain.
// @Description Delete a existing domain.
// @Produce json
// @Success 200
// @Router /v1.0/domains/{id} [delete]
func domainsIDDELETE(c *gin.Context) {

	// get id
	id := uuid.FromStringOrNil(c.Params.ByName("id"))

	tmp, exists := c.Get("user")
	if exists != true {
		logrus.Errorf("Could not find user info.")
		c.AbortWithStatus(400)
		return
	}
	u := tmp.(models.User)
	log := logrus.WithFields(logrus.Fields{
		"id":         u.ID,
		"username":   u.Username,
		"permission": u.Permission,
	})

	// delete a domain
	serviceHandler := c.MustGet(models.OBJServiceHandler).(servicehandler.ServiceHandler)
	if err := serviceHandler.DomainDelete(&u, id); err != nil {
		log.Errorf("Could not create a domain. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.AbortWithStatus(200)
	return
}

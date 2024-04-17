package queues

import (
	qmqueue "monorepo/bin-queue-manager/models/queue"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/api/models/response"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

// queuesGET handles GET /queues request.
// It returns list of queues of the given customer.
//
//	@Summary		List qeueus
//	@Description	get queues of the customer
//	@Produce		json
//	@Param			page_size	query		int		false	"The size of results. Max 100"
//	@Param			page_token	query		string	false	"The token. tm_create"
//	@Success		200			{object}	response.BodyQueuesGET
//	@Router			/v1.0/queues [get]
func queuesGET(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuesGET",
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

	var req request.ParamQueuesGET
	if err := c.BindQuery(&req); err != nil {
		c.AbortWithStatus(400)
		return
	}
	log.Debugf("Received request detail. page_size: %d, page_token: %s", req.PageSize, req.PageToken)

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// set max page size
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
		log.Debugf("Invalid requested page size. Set to default. page_size: %d", pageSize)
	} else if pageSize > 100 {
		pageSize = 100
		log.Debugf("Invalid requested page size. Set to max. page_size: %d", pageSize)
	}

	// get tmps
	tmps, err := serviceHandler.QueueGets(c.Request.Context(), &a, pageSize, req.PageToken)
	if err != nil {
		logrus.Errorf("Could not get queues info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	nextToken := ""
	if len(tmps) > 0 {
		nextToken = tmps[len(tmps)-1].TMCreate
	}
	res := response.BodyQueuesGET{
		Result: tmps,
		Pagination: response.Pagination{
			NextPageToken: nextToken,
		},
	}

	c.JSON(200, res)
}

// queuesPOST handles POST /queues request.
// It creates a new queue.
//
//	@Summary		Create a new queue.
//	@Description	create a new queue
//	@Produce		json
//	@Param			agent	body		request.BodyAgentsPOST	true	"The queue detail"
//	@Success		200		{object}	queue.WebhookMessage
//	@Router			/v1.0/queues [post]
func queuesPOST(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuesPOST",
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

	var req request.BodyQueuesPOST
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not parse the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// create
	res, err := serviceHandler.QueueCreate(
		c.Request.Context(),
		&a,
		req.Name,
		req.Detail,
		req.RoutingMethod,
		req.TagIDs,
		req.WaitActions,
		req.WaitTimeout,
		req.ServiceTimeout,
	)
	if err != nil {
		log.Errorf("Could not create a queue. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// queuesIDDelete handles DELETE /queues/<queue-id> request.
// It deletes the queue.
//
//	@Summary		Delete the queue
//	@Description	Delete the queue of the given id
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the queue"
//	@Success		200
//	@Router			/v1.0/queues/{id} [delete]
func queuesIDDelete(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuesIDDelete",
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
	if id == uuid.Nil {
		log.Error("Could not parse the id.")
		c.AbortWithStatus(400)
		return
	}

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// delete
	res, err := serviceHandler.QueueDelete(c.Request.Context(), &a, id)
	if err != nil {
		log.Infof("Could not get the delete the queue info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// queuesIDGet handles GET /queues/<queue-id> request.
// It gets the queue.
//
//	@Summary		Get the queue
//	@Description	Get the queue of the given id
//	@Produce		json
//	@Param			id	path	string	true	"The ID of the queue"
//	@Success		200
//	@Router			/v1.0/queues/{id} [get]
func queuesIDGet(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuesIDGet",
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

	// get service
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)

	// get
	res, err := serviceHandler.QueueGet(c.Request.Context(), &a, id)
	if err != nil {
		log.Infof("Could not get the queue info. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// queuesIDPUT handles PUT /queues/{id} request.
// It updates a queue basic info with the given info.
// And returns updated queue info if it succeed.
//
//	@Summary		Update an queue and reuturns updated queue info.
//	@Description	Update an queue and returns detail updated queue info.
//	@Produce		json
//	@Param			id			path	string					true	"The ID of the queue"
//	@Param			update_info	body	request.BodyQueuesIDPUT	true	"Queue's update info"
//	@Success		200
//	@Router			/v1.0/queues/{id} [put]
func queuesIDPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuesIDPUT",
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
	log = log.WithField("queue_id", id)
	log.Debug("Executing queuesIDPUT.")

	var req request.BodyQueuesIDPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the agent
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.QueueUpdate(c.Request.Context(), &a, id, req.Name, req.Detail, req.RoutingMethod, req.TagIDs, req.WaitActions, req.WaitTimeout, req.ServiceTimeout)
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// queuesIDTagIDsPUT handles PUT /queues/{id}/tag_ids request.
// It updates a queue's tag_ids info with the given info.
// And returns error if it failed.
//
//	@Summary		Update an queue's tag_ids info.
//	@Description	Update the queue's tag_ids.
//	@Produce		json
//	@Param			id			path	string							true	"The ID of the queue"
//	@Param			update_info	body	request.BodyQueuesIDTagIDsPUT	true	"Queue's update info"
//	@Success		200
//	@Router			/v1.0/queues/{id}/tag_ids [put]
func queuesIDTagIDsPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuesIDTagIDsPUT",
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
	log = log.WithField("queue_id", id)
	log.Debug("Executing queuesIDTagIDsPUT.")

	var req request.BodyQueuesIDTagIDsPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the queue
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.QueueUpdateTagIDs(c.Request.Context(), &a, id, req.TagIDs)
	if err != nil {
		log.Errorf("Could not update the agent. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// queuesIDRoutingMethodPUT handles PUT /queues/{id}/routing_method request.
// It updates a queue's routing_method info with the given info.
//
//	@Summary		Update an queue's tag_id info.
//	@Description	Update an queue routing_method info.
//	@Produce		json
//	@Param			id			path	string							true	"The ID of the queue"
//	@Param			update_info	body	request.BodyAgentsIDTagIDsPUT	true	"Queue's update info"
//	@Success		200
//	@Router			/v1.0/queues/{id}/routing_method [put]
func queuesIDRoutingMethodPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuesIDRoutingMethodPUT",
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
	log = log.WithField("queue_id", id)
	log.Debug("Executing queuesIDRoutingMethodPUT.")

	var req request.BodyQueuesIDRoutingMethodPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the queue
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.QueueUpdateRoutingMethod(c.Request.Context(), &a, id, qmqueue.RoutingMethod(req.RoutingMethod))
	if err != nil {
		log.Errorf("Could not update the queue. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

// queuesIDActionsPUT handles PUT /queues/{id}/actions request.
// It updates a queue's action hadnle with the given info.
//
//	@Summary		Update an queue's action handle info.
//	@Description	Update the queue's action handle info.
//	@Produce		json
//	@Param			id			path	string							true	"The ID of the queue"
//	@Param			update_info	body	request.BodyQueuesIDActionsPUT	true	"Queue's update info"
//	@Success		200
//	@Router			/v1.0/queues/{id}/status [put]
func queuesIDActionsPUT(c *gin.Context) {
	log := logrus.WithFields(logrus.Fields{
		"func":            "queuesIDActionsPUT",
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
	log = log.WithField("queue_id", id)
	log.Debug("Executing queuesIDActionsPUT.")

	var req request.BodyQueuesIDActionsPUT
	if err := c.BindJSON(&req); err != nil {
		log.Errorf("Could not bind the request. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	// update the queue
	serviceHandler := c.MustGet(common.OBJServiceHandler).(servicehandler.ServiceHandler)
	res, err := serviceHandler.QueueUpdateActions(c.Request.Context(), &a, id, req.WaitActions, req.TimeoutWait, req.TimeoutService)
	if err != nil {
		log.Errorf("Could not update the queue's action handle. err: %v", err)
		c.AbortWithStatus(400)
		return
	}

	c.JSON(200, res)
}

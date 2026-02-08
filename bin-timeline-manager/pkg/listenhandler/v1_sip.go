package listenhandler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-timeline-manager/pkg/listenhandler/models/request"
)

func (h *listenHandler) v1SIPInfoPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SIPInfoPost",
	})
	log.Info("RPC handler called - SIP info request received")

	// Parse request
	var req request.V1SIPInfoPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	// Parse time range
	fromTime, err := time.Parse(time.RFC3339, req.FromTime)
	if err != nil {
		log.Errorf("Could not parse from_time. err: %v", err)
		return simpleResponse(400), nil
	}

	toTime, err := time.Parse(time.RFC3339, req.ToTime)
	if err != nil {
		log.Errorf("Could not parse to_time. err: %v", err)
		return simpleResponse(400), nil
	}

	// Call handler
	result, err := h.sipHandler.GetSIPInfo(ctx, req.SIPCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get SIP info. err: %v", err)
		return simpleResponse(500), nil
	}

	// Marshal response
	data, err := json.Marshal(result)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal response")
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/json",
		Data:       data,
	}, nil
}

func (h *listenHandler) v1SIPPcapPost(ctx context.Context, m *sock.Request) (*sock.Response, error) {
	log := logrus.WithFields(logrus.Fields{
		"func": "v1SIPPcapPost",
	})
	log.Info("RPC handler called - PCAP request received")

	// Parse request
	var req request.V1SIPPcapPost
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Errorf("Could not unmarshal request. err: %v", err)
		return simpleResponse(400), nil
	}

	// Parse time range
	fromTime, err := time.Parse(time.RFC3339, req.FromTime)
	if err != nil {
		log.Errorf("Could not parse from_time. err: %v", err)
		return simpleResponse(400), nil
	}

	toTime, err := time.Parse(time.RFC3339, req.ToTime)
	if err != nil {
		log.Errorf("Could not parse to_time. err: %v", err)
		return simpleResponse(400), nil
	}

	// Call handler
	pcapData, err := h.sipHandler.GetPcap(ctx, req.SIPCallID, fromTime, toTime)
	if err != nil {
		log.Errorf("Could not get PCAP data. err: %v", err)
		return simpleResponse(500), nil
	}

	// Base64 encode PCAP data for JSON transport
	encoded := base64.StdEncoding.EncodeToString(pcapData)

	// Wrap in JSON object
	respData, err := json.Marshal(map[string]string{"data": encoded})
	if err != nil {
		log.Errorf("Could not marshal PCAP response. err: %v", err)
		return simpleResponse(500), nil
	}

	return &sock.Response{
		StatusCode: 200,
		DataType:   "application/octet-stream",
		Data:       respData,
	}, nil
}

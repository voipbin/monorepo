package requesthandler

import (
	"context"
	"encoding/json"

	"monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/models/sock"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
	cfrequest "monorepo/bin-conference-manager/pkg/listenhandler/models/request"

	"github.com/gofrs/uuid"
)

// ConferenceV1ServiceTypeConferencecallStart sends a request to chat-manager
// to starts a conferencecall service.
// it returns created service if it succeed.
func (r *requestHandler) ConferenceV1ServiceTypeConferencecallStart(
	ctx context.Context,
	activeflowID uuid.UUID,
	conferenceID uuid.UUID,
	referenceType cfconferencecall.ReferenceType,
	referenceID uuid.UUID,
) (*service.Service, error) {
	uri := "/v1/services/type/conferencecall"

	data := &cfrequest.V1DataServicesTypeConferencecallPost{
		ActiveflowID:  activeflowID,
		ConferenceID:  conferenceID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
	}

	m, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	tmp, err := r.sendRequestConference(ctx, uri, sock.RequestMethodPost, "conference/services/type/conferencecall", requestTimeoutDefault, 0, ContentTypeJSON, m)
	if err != nil {
		return nil, err
	}

	var res service.Service
	if errParse := parseResponse(tmp, &res); errParse != nil {
		return nil, errParse
	}

	return &res, nil
}

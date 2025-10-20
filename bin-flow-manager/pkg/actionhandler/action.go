package actionhandler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-flow-manager/models/action"
)

// ValidateActions validates actions
func (h *actionHandler) ValidateActions(actions []action.Action) error {

	for _, a := range actions {
		found := false
		for _, at := range action.TypeListAll {
			if a.Type == at {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("no support action type. type: %s", a.Type)
		}
	}

	return nil
}

// ActionFetchGet gets the action from the remote.
func (h *actionHandler) ActionFetchGet(act *action.Action, activeflowID uuid.UUID, referenceID uuid.UUID) ([]action.Action, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "ActionFetchGet",
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})

	tmpOption, err := json.Marshal(act.Option)
	if err != nil {
		return nil, errors.Wrapf(err, "could not marshal the option. action_id: %s", act.ID)
	}

	var option action.OptionFetch
	if err := json.Unmarshal(tmpOption, &option); err != nil {
		log.Errorf("Could not unmarshal the option. err: %v", err)
		return nil, err
	}

	// create a request body
	reqBody, err := json.Marshal(map[string]interface{}{
		"activeflow_id": activeflowID,
		"reference_id":  referenceID,
	})
	if err != nil {
		log.Errorf("Could not create a request body. err: %v", err)
		return nil, err
	}

	// set the HTTP method, url, and request body
	req, err := http.NewRequest(option.EventMethod, option.EventURL, bytes.NewBuffer(reqBody))
	if err != nil {
		log.Errorf("Could not create a request. err: %v", err)
		return nil, err
	}

	// set the request header Content-Type for json
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Could not get correct result. err: %v", err)
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("wrong status return. stauts: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Could not get the result. err: %v", err)
		return nil, err
	}

	var res []action.Action
	if err := json.Unmarshal(body, &res); err != nil {
		log.Errorf("Could not unmarshal the result. body: %s, err: %v", body, err)
		return nil, err
	}

	return res, nil
}

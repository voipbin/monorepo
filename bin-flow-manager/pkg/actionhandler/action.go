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

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
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
			return cerrors.InvalidArgument(
				commonoutline.ServiceNameFlowManager,
				"INVALID_ACTION_TYPE",
				fmt.Sprintf("not supported action type: %s", a.Type),
			)
		}

		// validate anonymous option for call and connect actions
		switch a.Type {
		case action.TypeCall:
			var opt action.OptionCall
			if err := action.ParseOption(a.Option, &opt); err != nil {
				logrus.WithField("action", a).Warnf("Could not parse call action option for anonymous validation. err: %v", err)
			} else if opt.Anonymous != "" {
				if !action.ValidateAnonymous(opt.Anonymous) {
					return cerrors.InvalidArgument(
						commonoutline.ServiceNameFlowManager,
						"INVALID_ANONYMOUS_VALUE",
						fmt.Sprintf("invalid anonymous value for call action: %s", opt.Anonymous),
					)
				}
			}
		case action.TypeConnect:
			var opt action.OptionConnect
			if err := action.ParseOption(a.Option, &opt); err != nil {
				logrus.WithField("action", a).Warnf("Could not parse connect action option for anonymous validation. err: %v", err)
			} else if opt.Anonymous != "" {
				if !action.ValidateAnonymous(opt.Anonymous) {
					return cerrors.InvalidArgument(
						commonoutline.ServiceNameFlowManager,
						"INVALID_ANONYMOUS_VALUE",
						fmt.Sprintf("invalid anonymous value for connect action: %s", opt.Anonymous),
					)
				}
			}
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

package flowhandler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

// actionPatchGet gets the action from the remote.
func (h *flowHandler) actionPatchGet(act *action.Action, callID uuid.UUID) ([]action.Action, error) {

	var option action.OptionPatch
	if err := json.Unmarshal(act.Option, &option); err != nil {
		logrus.Errorf("Could not unmarshal the option. err: %v", err)
		return nil, err
	}

	// create a request body
	reqBody, err := json.Marshal(map[string]interface{}{
		"call_id": callID.String(),
	})
	if err != nil {
		logrus.Errorf("Could not create a request body. err: %v", err)
		return nil, err
	}

	// set the HTTP method, url, and request body
	req, err := http.NewRequest(option.EventMethod, option.EventURL, bytes.NewBuffer(reqBody))
	if err != nil {
		logrus.Errorf("Could not create a request. err: %v", err)
		return nil, err
	}

	// set the request header Content-Type for json
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("Could not get correct result. err: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("wrong status return. stauts: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Could not get the result. err: %v", err)
		return nil, err
	}

	var res []action.Action
	if err := json.Unmarshal(body, &res); err != nil {
		logrus.Errorf("Could not unmarshal the result. body: %s, err: %v", body, err)
		return nil, err
	}

	return res, nil
}

// ActionGet returns corresponded action.
func (h *flowHandler) ActionGet(ctx context.Context, flowID uuid.UUID, actionID uuid.UUID) (*action.Action, error) {
	flow, err := h.FlowGet(ctx, flowID)
	if err != nil {
		return nil, err
	}

	for _, action := range flow.Actions {
		if action.ID == actionID {
			return &action, nil
		}
	}

	return nil, dbhandler.ErrNotFound
}

func (h *flowHandler) CreateActionHangup() *action.Action {

	opt := action.OptionHangup{}

	optString, err := json.Marshal(opt)
	if err != nil {
		logrus.Errorf("Could not marshal the hangup option. err: %v", err)
		return nil
	}

	res := action.Action{
		ID:     action.IDFinish,
		Type:   action.TypeHangup,
		Option: optString,
	}

	return &res
}

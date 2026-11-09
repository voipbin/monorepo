package pushringhandler

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	commonaddress "monorepo/bin-common-handler/models/address"
	pmnotification "monorepo/bin-push-manager/models/notification"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-call-manager/models/pushring"
)

const (
	directPrefix = "direct."
	ticketBytes  = 16
	// ticketLen is the length of a ticket as it arrives in the SIP request: the direct prefix and 16 random bytes
	// in hex. The direct hashes of the direct manager are shorter, so this length alone keeps ordinary direct calls
	// away from the ticket lookup.
	ticketLen = len(directPrefix) + ticketBytes*2

	notificationTimeout = 10 * time.Second
	agentLookupTimeout  = 1000 // ms
)

// startRequest is the input of one start.
type startRequest struct {
	customerID   uuid.UUID
	destination  commonaddress.Address
	confbridgeID uuid.UUID
	masterCallID uuid.UUID
	flowID       uuid.UUID
	source       commonaddress.Address
}

// Start starts a session for the call to the extension.
//
// Every failure of the lookups means "no session": the result says started false and the caller continues with the
// ordinary dial. Sending the notifications is not part of the start; it runs after the result is returned.
func (h *pushRingHandler) Start(
	ctx context.Context,
	customerID uuid.UUID,
	destination commonaddress.Address,
	confbridgeID uuid.UUID,
	masterCallID uuid.UUID,
	flowID uuid.UUID,
	source commonaddress.Address,
) (*pushring.StartResult, error) {
	req := &startRequest{
		customerID:   customerID,
		destination:  destination,
		confbridgeID: confbridgeID,
		masterCallID: masterCallID,
		flowID:       flowID,
		source:       source,
	}

	log := logrus.WithFields(logrus.Fields{
		"func":           "Start",
		"customer_id":    req.customerID,
		"confbridge_id":  req.confbridgeID,
		"master_call_id": req.masterCallID,
	})

	ctx, cancel := context.WithTimeout(ctx, h.startTimeout)
	defer cancel()

	notStarted := &pushring.StartResult{}

	if req.destination.Type != commonaddress.TypeExtension {
		log.Debugf("The destination is not an extension. type: %s", req.destination.Type)
		return notStarted, nil
	}

	// the contact lookup runs in parallel with the chain extension -> agent -> devices.
	type contactResult struct {
		registered bool
	}
	contactCh := make(chan contactResult, 1)
	go func() {
		contactCh <- contactResult{registered: h.sipRegistered(ctx, req)}
	}()

	extensionID, err := h.getExtensionID(ctx, req)
	if err != nil {
		log.Infof("Could not resolve the extension. err: %v", err)
		return notStarted, nil
	}

	agentID, err := h.getAgentID(ctx, req.customerID, extensionID)
	if err != nil {
		log.Infof("Could not resolve the agent of the extension. extension_id: %s, err: %v", extensionID, err)
		return notStarted, nil
	}

	devices, err := h.reqHandler.PushV1DeviceList(ctx, req.customerID, agentID)
	if err != nil {
		log.Infof("Could not list the devices of the agent. agent_id: %s, err: %v", agentID, err)
		return notStarted, nil
	}
	if len(devices) == 0 {
		log.Debugf("The agent has no device. agent_id: %s", agentID)
		return notStarted, nil
	}

	var contact contactResult
	select {
	case contact = <-contactCh:
	case <-ctx.Done():
		log.Infof("The start ran out of time. err: %v", ctx.Err())
		return notStarted, nil
	}

	now := h.now()
	s := &pushring.Session{
		ID:            h.utilHandler.UUIDCreate(),
		CustomerID:    req.customerID,
		AgentID:       agentID,
		ExtensionID:   extensionID,
		ConfbridgeID:  req.confbridgeID,
		MasterCallID:  req.masterCallID,
		FlowID:        req.flowID,
		ExpiresAt:     now.Add(pushring.Window),
		State:         pushring.StateRinging,
		SIPRegistered: contact.registered,
		Tickets:       map[string]uuid.UUID{},
	}

	// one ticket per device. The value of the ticket is given to that device only; the cache holds its hash.
	tickets := make(map[uuid.UUID]string, len(devices))
	for _, d := range devices {
		ticket, errTicket := generateTicket()
		if errTicket != nil {
			log.Errorf("Could not generate a ticket. err: %v", errTicket)
			return notStarted, nil
		}
		tickets[d.ID] = ticket
		s.Tickets[hashTicket(ticket)] = d.ID
	}

	if errCreate := h.cache.PushRingSessionCreate(ctx, s); errCreate != nil {
		log.Errorf("Could not create the session. err: %v", errCreate)
		return notStarted, nil
	}
	log = log.WithField("session_id", s.ID)

	if errFlag := h.addFlag(ctx, req.confbridgeID); errFlag != nil {
		log.Errorf("Could not mark the confbridge. err: %v", errFlag)
		// the call continues as before. Nothing has been sent, so only the keys are removed.
		// the cleanup does not depend on the request context, which may be what failed.
		cleanCtx := context.WithoutCancel(ctx)
		if closed, errClose := h.cache.PushRingClose(cleanCtx, s.ID); errClose == nil && closed != nil {
			_ = h.cache.PushRingTicketsDelete(cleanCtx, closed, uuid.Nil)
		}
		return notStarted, nil
	}

	// the window ends with the expire request. The second request is redundant, for the case the first is lost.
	for _, delay := range []time.Duration{pushring.Window, pushring.RedundantExpireDelay} {
		if errExpire := h.reqHandler.CallV1PushRingExpire(ctx, s.ID, int(delay/time.Millisecond)); errExpire != nil {
			log.Errorf("Could not schedule the expire request. delay: %s, err: %v", delay, errExpire)
		}
	}

	h.goAsync(ctx, func(ctx context.Context) { h.sendIncoming(ctx, s, req, tickets) })

	log.Debugf("Started the push ring session. device_count: %d, sip_registered: %v", len(devices), s.SIPRegistered)

	return &pushring.StartResult{
		Started:       true,
		SessionID:     s.ID,
		SIPRegistered: s.SIPRegistered,
	}, nil
}

// getExtensionID returns the id of the destination extension. The target of a connect destination is the extension id
// or, for a destination written with the number, the number (the number is then in the target name or the target).
func (h *pushRingHandler) getExtensionID(ctx context.Context, req *startRequest) (uuid.UUID, error) {
	if id := uuid.FromStringOrNil(req.destination.Target); id != uuid.Nil {
		return id, nil
	}

	number := req.destination.TargetName
	if number == "" {
		number = req.destination.Target
	}

	filters := map[rmextension.Field]any{
		rmextension.FieldCustomerID: req.customerID,
		rmextension.FieldExtension:  number,
		rmextension.FieldDeleted:    false,
	}
	res, err := h.reqHandler.RegistrarV1ExtensionList(ctx, "", 1, filters)
	if err != nil {
		return uuid.Nil, errors.Wrap(err, "could not list the extension")
	}
	if len(res) == 0 {
		return uuid.Nil, errors.New("the extension was not found")
	}

	return res[0].ID, nil
}

// getAgentID returns the id of the agent that owns the extension. The agent must belong to the customer and must not
// be deleted.
func (h *pushRingHandler) getAgentID(ctx context.Context, customerID, extensionID uuid.UUID) (uuid.UUID, error) {
	addr := commonaddress.Address{
		Type:   commonaddress.TypeExtension,
		Target: extensionID.String(),
	}
	ag, err := h.reqHandler.AgentV1AgentGetByCustomerIDAndAddress(ctx, agentLookupTimeout, customerID, addr)
	if err != nil {
		return uuid.Nil, errors.Wrap(err, "could not get the agent")
	}
	if ag == nil || ag.ID == uuid.Nil {
		return uuid.Nil, errors.New("no agent")
	}
	if ag.CustomerID != customerID {
		return uuid.Nil, errors.New("the agent belongs to another customer")
	}
	if ag.TMDelete != nil && ag.TMDelete.Year() < 9000 {
		return uuid.Nil, errors.New("the agent is deleted")
	}

	return ag.ID, nil
}

// sipRegistered returns true when the extension has a registered contact. A failed lookup counts as registered: the
// pushes alone must never end the call when the state of the registration is unknown.
func (h *pushRingHandler) sipRegistered(ctx context.Context, req *startRequest) bool {
	log := logrus.WithFields(logrus.Fields{
		"func":        "sipRegistered",
		"customer_id": req.customerID,
	})

	number := req.destination.TargetName
	if number == "" {
		id := uuid.FromStringOrNil(req.destination.Target)
		if id == uuid.Nil {
			number = req.destination.Target
		} else {
			ext, err := h.reqHandler.RegistrarV1ExtensionGet(ctx, id)
			if err != nil {
				log.Infof("Could not get the extension. err: %v", err)
				return true
			}
			number = ext.Extension
		}
	}

	contacts, err := h.reqHandler.RegistrarV1ContactList(ctx, map[string]any{
		"customer_id": req.customerID,
		"extension":   number,
	})
	if err != nil {
		log.Infof("Could not list the contacts. err: %v", err)
		return true
	}

	return len(contacts) > 0
}

// addFlag marks the confbridge as the confbridge of a push session. Only a marked confbridge reads the session on a join.
func (h *pushRingHandler) addFlag(ctx context.Context, id uuid.UUID) error {
	cb, err := h.db.ConfbridgeGet(ctx, id)
	if err != nil {
		return errors.Wrap(err, "could not get the confbridge")
	}

	if slices.Contains(cb.Flags, confbridge.FlagPushRing) {
		return nil
	}

	flags := append(slices.Clone(cb.Flags), confbridge.FlagPushRing)
	if errSet := h.db.ConfbridgeSetFlags(ctx, id, flags); errSet != nil {
		return errors.Wrap(errSet, "could not set the flags")
	}

	return nil
}

// sendIncoming sends the incoming notification to every device, each with its own join address. When every device
// answers with an explicit failure, the pushes are given up. An error or a timeout is not a failure: the notification
// may have been delivered.
func (h *pushRingHandler) sendIncoming(ctx context.Context, s *pushring.Session, req *startRequest, tickets map[uuid.UUID]string) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "sendIncoming",
		"session_id": s.ID,
	})

	display := req.source.TargetName
	if display == "" {
		display = req.source.Target
	}

	var mu sync.Mutex
	failed := 0

	var wg sync.WaitGroup
	for deviceID, ticket := range tickets {
		wg.Add(1)
		go func(deviceID uuid.UUID, ticket string) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(ctx, notificationTimeout)
			defer cancel()

			data := map[string]string{
				"call_id":        s.MasterCallID.String(),
				"caller_display": display,
				"join_address":   fmt.Sprintf(pushring.JoinAddressFormat, ticket),
				"expires_at":     s.ExpiresAt.UTC().Format(time.RFC3339),
				"cancel_key":     s.ID.String(),
			}

			res, err := h.reqHandler.PushV1NotificationSend(ctx, s.CustomerID, s.AgentID, deviceID, pmnotification.KindCallIncoming, data)
			if err != nil {
				log.Infof("Could not send the notification. The result is unknown. device_id: %s, err: %v", deviceID, err)
				return
			}
			if res.SentCount == 0 {
				mu.Lock()
				failed++
				mu.Unlock()
			}
		}(deviceID, ticket)
	}
	wg.Wait()

	if failed == len(tickets) {
		h.pushFailed(ctx, s)
	}
}

// pushFailed gives up the pushes of the session. The SIP gate stays open. Only when no SIP contact is registered the
// call is ended, because nothing can answer it.
func (h *pushRingHandler) pushFailed(ctx context.Context, s *pushring.Session) {
	log := logrus.WithFields(logrus.Fields{
		"func":       "pushFailed",
		"session_id": s.ID,
	})

	changed, err := h.cache.PushRingFailed(ctx, s.ID)
	if err != nil {
		log.Errorf("Could not move the session to push_failed. err: %v", err)
		return
	}
	if !changed {
		return
	}

	if errDelete := h.cache.PushRingTicketsDelete(ctx, s, uuid.Nil); errDelete != nil {
		log.Errorf("Could not delete the tickets. err: %v", errDelete)
	}

	if s.SIPRegistered {
		log.Infof("Every notification failed. The SIP legs continue.")
		return
	}

	log.Infof("Every notification failed and no SIP contact is registered. Hanging up the call.")
	if errHangup := h.hangupCall(ctx, s.MasterCallID, call.HangupReasonNoanswer); errHangup != nil {
		log.Errorf("Could not hang up the master call. err: %v", errHangup)
	}
}

func (h *pushRingHandler) hangupCall(ctx context.Context, id uuid.UUID, reason call.HangupReason) error {
	if h.hangup == nil {
		return errors.New("the hangup function is not set")
	}

	return h.hangup(ctx, id, reason)
}

// goAsync runs the function in a goroutine with a context that is not canceled with the request.
func (h *pushRingHandler) goAsync(ctx context.Context, f func(ctx context.Context)) {
	ctx = context.WithoutCancel(ctx)

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		f(ctx)
	}()
}

func generateTicket() (string, error) {
	b := make([]byte, ticketBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return directPrefix + hex.EncodeToString(b), nil
}

func hashTicket(ticket string) string {
	sum := sha256.Sum256([]byte(ticket))
	return hex.EncodeToString(sum[:])
}

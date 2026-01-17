package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/sirupsen/logrus"

	"monorepo/bin-talk-manager/pkg/messagehandler"
	"monorepo/bin-talk-manager/pkg/participanthandler"
	"monorepo/bin-talk-manager/pkg/reactionhandler"
	"monorepo/bin-talk-manager/pkg/talkhandler"
	commonoutline "monorepo/bin-common-handler/models/outline"
	commonsock "monorepo/bin-common-handler/models/sock"
	commonsockhandler "monorepo/bin-common-handler/pkg/sockhandler"
)

// Regex patterns for URI matching
var (
	regV1Chats                 = regexp.MustCompile(`^/v1/chats$`)
	regV1ChatsID               = regexp.MustCompile(`^/v1/chats/([^/]+)$`)
	regV1ChatsIDParticipants   = regexp.MustCompile(`^/v1/chats/([^/]+)/participants$`)
	regV1ChatsIDParticipantsID = regexp.MustCompile(`^/v1/chats/([^/]+)/participants/([^/]+)$`)
	regV1Messages              = regexp.MustCompile(`^/v1/messages$`)
	regV1MessagesID            = regexp.MustCompile(`^/v1/messages/([^/]+)$`)
	regV1MessagesIDReactions   = regexp.MustCompile(`^/v1/messages/([^/]+)/reactions$`)
)

type listenHandler struct {
	sockHandler         commonsockhandler.SockHandler
	talkHandler         talkhandler.TalkHandler
	messageHandler      messagehandler.MessageHandler
	participantHandler  participanthandler.ParticipantHandler
	reactionHandler     reactionhandler.ReactionHandler
}

// New creates a new listen handler
func New(
	sock commonsockhandler.SockHandler,
	talk talkhandler.TalkHandler,
	msg messagehandler.MessageHandler,
	part participanthandler.ParticipantHandler,
	react reactionhandler.ReactionHandler,
) *listenHandler {
	return &listenHandler{
		sockHandler:         sock,
		talkHandler:         talk,
		messageHandler:      msg,
		participantHandler:  part,
		reactionHandler:     react,
	}
}

// Listen starts listening for RabbitMQ messages
func (h *listenHandler) Listen(ctx context.Context) error {
	// Create queue
	if err := h.sockHandler.QueueCreate(string(commonoutline.QueueNameTalkRequest), "normal"); err != nil {
		return err
	}

	// Start consuming RPC requests
	return h.sockHandler.ConsumeRPC(
		ctx,
		string(commonoutline.QueueNameTalkRequest),
		string(commonoutline.ServiceNameTalkManager),
		false, // exclusive
		false, // noLocal
		false, // noWait
		10,    // workers
		h.processRequest,
	)
}

// processRequest routes incoming requests to appropriate handlers
func (h *listenHandler) processRequest(m *commonsock.Request) (*commonsock.Response, error) {
	ctx := context.Background()
	logrus.Debugf("Received request: %s %s", m.Method, m.URI)

	var response *commonsock.Response
	var err error

	// Route based on URI pattern and HTTP method
	switch {
	// v1

	// chats
	case regV1Chats.MatchString(m.URI) && m.Method == commonsock.RequestMethodPost:
		response, err = h.v1ChatsPost(ctx, *m)

	// chats
	case regV1Chats.MatchString(m.URI) && m.Method == commonsock.RequestMethodGet:
		response, err = h.v1ChatsGet(ctx, *m)

	// chats/<chat-id>
	case regV1ChatsID.MatchString(m.URI) && m.Method == commonsock.RequestMethodGet:
		response, err = h.v1ChatsIDGet(ctx, *m)

	// chats/<chat-id>
	case regV1ChatsID.MatchString(m.URI) && m.Method == commonsock.RequestMethodDelete:
		response, err = h.v1ChatsIDDelete(ctx, *m)

	// chats/<chat-id>/participants
	case regV1ChatsIDParticipants.MatchString(m.URI) && m.Method == commonsock.RequestMethodPost:
		response, err = h.v1ChatsIDParticipantsPost(ctx, *m)

	// chats/<chat-id>/participants
	case regV1ChatsIDParticipants.MatchString(m.URI) && m.Method == commonsock.RequestMethodGet:
		response, err = h.v1ChatsIDParticipantsGet(ctx, *m)

	// chats/<chat-id>/participants/<participant-id>
	case regV1ChatsIDParticipantsID.MatchString(m.URI) && m.Method == commonsock.RequestMethodDelete:
		response, err = h.v1ChatsIDParticipantsIDDelete(ctx, *m)

	// messages
	case regV1Messages.MatchString(m.URI) && m.Method == commonsock.RequestMethodPost:
		response, err = h.v1MessagesPost(ctx, *m)

	// messages
	case regV1Messages.MatchString(m.URI) && m.Method == commonsock.RequestMethodGet:
		response, err = h.v1MessagesGet(ctx, *m)

	// messages/<message-id>
	case regV1MessagesID.MatchString(m.URI) && m.Method == commonsock.RequestMethodGet:
		response, err = h.v1MessagesIDGet(ctx, *m)

	// messages/<message-id>
	case regV1MessagesID.MatchString(m.URI) && m.Method == commonsock.RequestMethodDelete:
		response, err = h.v1MessagesIDDelete(ctx, *m)

	// messages/<message-id>/reactions
	case regV1MessagesIDReactions.MatchString(m.URI) && m.Method == commonsock.RequestMethodPost:
		response, err = h.v1MessagesIDReactionsPost(ctx, *m)

	// messages/<message-id>/reactions
	case regV1MessagesIDReactions.MatchString(m.URI) && m.Method == commonsock.RequestMethodDelete:
		response, err = h.v1MessagesIDReactionsDelete(ctx, *m)

	default:
		logrus.Warnf("Unknown URI or method: %s %s", m.Method, m.URI)
		return simpleResponse(404), nil
	}

	if err != nil {
		logrus.Errorf("Request failed: %v", err)
		return simpleResponse(500), err
	}

	return response, nil
}

// simpleResponse creates a simple response with status code
func simpleResponse(statusCode int) *commonsock.Response {
	return &commonsock.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       json.RawMessage("{}"),
	}
}

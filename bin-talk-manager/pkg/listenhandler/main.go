package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"regexp"

	log "github.com/sirupsen/logrus"

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
	regV1TalkChats                 = regexp.MustCompile(`^/v1/chats$`)
	regV1TalkChatsID               = regexp.MustCompile(`^/v1/chats/([^/]+)$`)
	regV1TalkChatsIDParticipants   = regexp.MustCompile(`^/v1/chats/([^/]+)/participants$`)
	regV1TalkChatsIDParticipantsID = regexp.MustCompile(`^/v1/chats/([^/]+)/participants/([^/]+)$`)
	regV1TalkMessages              = regexp.MustCompile(`^/v1/messages$`)
	regV1TalkMessagesID            = regexp.MustCompile(`^/v1/messages/([^/]+)$`)
	regV1TalkMessagesIDReactions   = regexp.MustCompile(`^/v1/messages/([^/]+)/reactions$`)
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
	log.Debugf("Received request: %s %s", m.Method, m.URI)

	// Route based on URI pattern
	switch {
	case regV1TalkChats.MatchString(m.URI):
		return h.processV1TalkChats(ctx, *m)
	case regV1TalkChatsID.MatchString(m.URI):
		return h.processV1TalkChatsID(ctx, *m)
	case regV1TalkChatsIDParticipants.MatchString(m.URI):
		return h.processV1TalkChatsIDParticipants(ctx, *m)
	case regV1TalkChatsIDParticipantsID.MatchString(m.URI):
		return h.processV1TalkChatsIDParticipantsID(ctx, *m)
	case regV1TalkMessages.MatchString(m.URI):
		return h.processV1TalkMessages(ctx, *m)
	case regV1TalkMessagesID.MatchString(m.URI):
		return h.processV1TalkMessagesID(ctx, *m)
	case regV1TalkMessagesIDReactions.MatchString(m.URI):
		return h.processV1TalkMessagesIDReactions(ctx, *m)
	default:
		log.Warnf("Unknown URI: %s", m.URI)
		return simpleResponse(404), nil
	}
}

// simpleResponse creates a simple response with status code
func simpleResponse(statusCode int) *commonsock.Response {
	return &commonsock.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       json.RawMessage("{}"),
	}
}

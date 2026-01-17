package listenhandler

import (
	"context"
	"regexp"

	log "github.com/sirupsen/logrus"

	"monorepo/bin-talk-manager/pkg/messagehandler"
	"monorepo/bin-talk-manager/pkg/participanthandler"
	"monorepo/bin-talk-manager/pkg/reactionhandler"
	"monorepo/bin-talk-manager/pkg/talkhandler"
	commonsock "monorepo/bin-common-handler/models/sock"
	commonsockhandler "monorepo/bin-common-handler/pkg/sockhandler"
)

// Regex patterns for URI matching
var (
	regV1Talks            = regexp.MustCompile(`^/v1/talks$`)
	regV1TalksID          = regexp.MustCompile(`^/v1/talks/([^/]+)$`)
	regV1TalksIDParticipants = regexp.MustCompile(`^/v1/talks/([^/]+)/participants$`)
	regV1TalksIDParticipantsID = regexp.MustCompile(`^/v1/talks/([^/]+)/participants/([^/]+)$`)
	regV1Messages         = regexp.MustCompile(`^/v1/messages$`)
	regV1MessagesID       = regexp.MustCompile(`^/v1/messages/([^/]+)$`)
	regV1MessagesIDReactions = regexp.MustCompile(`^/v1/messages/([^/]+)/reactions$`)
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
func (h *listenHandler) Listen(ctx context.Context) {
	h.sockHandler.SubscribeSync(ctx, h.processRequest)
}

// processRequest routes incoming requests to appropriate handlers
func (h *listenHandler) processRequest(ctx context.Context, m commonsock.Request) (commonsock.Response, error) {
	log.Debugf("Received request: %s %s", m.Method, m.URI)

	// Route based on URI pattern
	switch {
	case regV1Talks.MatchString(m.URI):
		return h.processV1Talks(ctx, m)
	case regV1TalksID.MatchString(m.URI):
		return h.processV1TalksID(ctx, m)
	case regV1TalksIDParticipants.MatchString(m.URI):
		return h.processV1TalksIDParticipants(ctx, m)
	case regV1TalksIDParticipantsID.MatchString(m.URI):
		return h.processV1TalksIDParticipantsID(ctx, m)
	case regV1Messages.MatchString(m.URI):
		return h.processV1Messages(ctx, m)
	case regV1MessagesID.MatchString(m.URI):
		return h.processV1MessagesID(ctx, m)
	case regV1MessagesIDReactions.MatchString(m.URI):
		return h.processV1MessagesIDReactions(ctx, m)
	default:
		log.Warnf("Unknown URI: %s", m.URI)
		return simpleResponse(404), nil
	}
}

// simpleResponse creates a simple response with status code
func simpleResponse(statusCode int) commonsock.Response {
	return commonsock.Response{
		StatusCode: statusCode,
		DataType:   "application/json",
		Data:       "{}",
	}
}

package listenhandler

//go:generate mockgen -package listenhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-chat-manager/models/chat"
	"monorepo/bin-chat-manager/models/chatroom"
	"monorepo/bin-chat-manager/models/messagechat"
	"monorepo/bin-chat-manager/models/messagechatroom"
	"monorepo/bin-chat-manager/pkg/chathandler"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/messagechathandler"
	"monorepo/bin-chat-manager/pkg/messagechatroomhandler"
)

// pagination parameters
const (
	PageSize  = "page_size"
	PageToken = "page_token"
)

// ListenHandler interface
type ListenHandler interface {
	Run(queue, exchangeDelay string) error
}

type listenHandler struct {
	sockHandler sockhandler.SockHandler

	chatHandler     chathandler.ChatHandler
	chatroomHandler chatroomhandler.ChatroomHandler

	messagechatHandler     messagechathandler.MessagechatHandler
	messagechatroomHandler messagechatroomhandler.MessagechatroomHandler
}

var (
	regUUID = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

	// chats
	regV1Chats                   = regexp.MustCompile("/v1/chats$")
	regV1ChatsGet                = regexp.MustCompile(`/v1/chats\?`)
	regV1ChatsID                 = regexp.MustCompile("/v1/chats/" + regUUID + "$")
	regV1ChatsIDRoomOwnerID      = regexp.MustCompile("/v1/chats/" + regUUID + "/room_owner_id$")
	regV1ChatsIDParticipantIDs   = regexp.MustCompile("/v1/chats/" + regUUID + "/participant_ids$")
	regV1ChatsIDParticipantIDsID = regexp.MustCompile("/v1/chats/" + regUUID + "/participant_ids/" + regUUID + "$")

	// chatrooms
	regV1ChatroomsGet = regexp.MustCompile(`/v1/chatrooms\?`)
	regV1ChatroomsID  = regexp.MustCompile("/v1/chatrooms/" + regUUID + "$")

	// messagechats
	regV1Messagechats    = regexp.MustCompile("/v1/messagechats$")
	regV1MessagechatsGet = regexp.MustCompile(`/v1/messagechats\?`)
	regV1MessagechatsID  = regexp.MustCompile("/v1/messagechats/" + regUUID + "$")

	// messagechatrooms
	regV1MessagechatroomsGet = regexp.MustCompile(`/v1/messagechatrooms\?`)
	regV1MessagechatroomsID  = regexp.MustCompile("/v1/messagechatrooms/" + regUUID + "$")
)

var (
	metricsNamespace = "chat_manager"

	promReceivedRequestProcessTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "receive_request_process_time",
			Help:      "Process time of received request",
			Buckets: []float64{
				50, 100, 500, 1000, 3000,
			},
		},
		[]string{"type", "method"},
	)
)

func init() {
	prometheus.MustRegister(
		promReceivedRequestProcessTime,
	)
}

// simpleResponse returns simple rabbitmq response
func simpleResponse(code int) *sock.Response {
	return &sock.Response{
		StatusCode: code,
	}
}

// NewListenHandler return ListenHandler interface
func NewListenHandler(
	sockHandler sockhandler.SockHandler,

	chatHandler chathandler.ChatHandler,
	chatroomHandler chatroomhandler.ChatroomHandler,
	messagechatHandler messagechathandler.MessagechatHandler,
	messagechatroomHandler messagechatroomhandler.MessagechatroomHandler,
) ListenHandler {
	h := &listenHandler{
		sockHandler: sockHandler,

		chatHandler:     chatHandler,
		chatroomHandler: chatroomHandler,

		messagechatHandler:     messagechatHandler,
		messagechatroomHandler: messagechatroomHandler,
	}

	return h
}

func (h *listenHandler) Run(queue, exchangeDelay string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Run",
		"queue":         queue,
		"exchage_delay": exchangeDelay,
	})
	log.Info("Creating rabbitmq queue for listen.")

	if err := h.sockHandler.QueueCreate(queue, "normal"); err != nil {
		return fmt.Errorf("could not declare the queue for listenHandler. err: %v", err)
	}

	// process the received request
	go func() {
		if errConsume := h.sockHandler.ConsumeRPC(context.Background(), queue, string(commonoutline.ServiceNameChatManager), false, false, false, 10, h.processRequest); errConsume != nil {
			log.Errorf("Could not consume the request message correctly. err: %v", errConsume)
		}
	}()

	return nil
}

func (h *listenHandler) processRequest(m *sock.Request) (*sock.Response, error) {

	var requestType string
	var err error
	var response *sock.Response

	ctx := context.Background()

	logrus.WithFields(logrus.Fields{
		"uri":       m.URI,
		"method":    m.Method,
		"data_type": m.DataType,
		"data":      m.Data,
	}).Debugf("Received request. method: %s, uri: %s", m.Method, m.URI)

	start := time.Now()
	switch {

	// v1
	// chats
	case regV1ChatsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/chats"
		response, err = h.v1ChatsGet(ctx, m)

	case regV1Chats.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/chats"
		response, err = h.v1ChatsPost(ctx, m)

	// chats/<chat-id>
	case regV1ChatsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/chats/<chat-id>"
		response, err = h.v1ChatsIDGet(ctx, m)

	case regV1ChatsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/chats/<chat-id>"
		response, err = h.v1ChatsIDDelete(ctx, m)

	case regV1ChatsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/chats/<chat-id>"
		response, err = h.v1ChatsIDPut(ctx, m)

	// chats/<chat-id>/room_owner_id
	case regV1ChatsIDRoomOwnerID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/chats/<chat-id>/room_owner_id"
		response, err = h.v1ChatsIDRoomOwnerIDPut(ctx, m)

	// chats/<chat-id>/participant_ids
	case regV1ChatsIDParticipantIDs.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/chats/<chat-id>/participant_ids"
		response, err = h.v1ChatsIDParticipantIDsPost(ctx, m)

	// chats/<chat-id>/participant_ids/<participant-id>
	case regV1ChatsIDParticipantIDsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/chats/<chat-id>/participant_ids/<participant-id>"
		response, err = h.v1ChatsIDParticipantIDsIDDelete(ctx, m)

	// chatrooms
	case regV1ChatroomsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/chatrooms"
		response, err = h.v1ChatroomsGet(ctx, m)

	// chatrooms/<chatroom-id>
	case regV1ChatroomsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/chatrooms/<chatroom-id>"
		response, err = h.v1ChatroomsIDGet(ctx, m)

	case regV1ChatroomsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/chatrooms/<chatroom-id>"
		response, err = h.v1ChatroomsIDDelete(ctx, m)

	case regV1ChatroomsID.MatchString(m.URI) && m.Method == sock.RequestMethodPut:
		requestType = "/chatrooms/<chatroom-id>"
		response, err = h.v1ChatroomsIDPut(ctx, m)

	// messagechats
	case regV1MessagechatsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/messagechats"
		response, err = h.v1MessagechatsGet(ctx, m)

	case regV1Messagechats.MatchString(m.URI) && m.Method == sock.RequestMethodPost:
		requestType = "/messagechats"
		response, err = h.v1MessagechatsPost(ctx, m)

	// messagechats/<messagechat-id>
	case regV1MessagechatsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/messagechats/<messagechat-id>"
		response, err = h.v1MessagechatsIDGet(ctx, m)

	case regV1MessagechatsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/messagechats/<messagechat-id>"
		response, err = h.v1MessagechatsIDDelete(ctx, m)

	// messagechatrooms
	case regV1MessagechatroomsGet.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/messagechatrooms"
		response, err = h.v1MessagechatroomsGet(ctx, m)

	// messagechatrooms/<messagechatroom-id>
	case regV1MessagechatroomsID.MatchString(m.URI) && m.Method == sock.RequestMethodGet:
		requestType = "/messagechatrooms/<messagechatroom-id>"
		response, err = h.v1MessagechatroomsIDGet(ctx, m)

	case regV1MessagechatroomsID.MatchString(m.URI) && m.Method == sock.RequestMethodDelete:
		requestType = "/messagechatrooms/<messagechatroom-id>"
		response, err = h.v1MessagechatroomsIDDelete(ctx, m)

	default:
		logrus.WithFields(logrus.Fields{
			"uri":    m.URI,
			"method": m.Method,
		}).Errorf("Could not find corresponded message handler. data: %s", m.Data)
		response = simpleResponse(404)
		err = nil
		requestType = "notfound"
	}
	elapsed := time.Since(start)
	promReceivedRequestProcessTime.WithLabelValues(requestType, string(m.Method)).Observe(float64(elapsed.Milliseconds()))

	// default error handler
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"uri":    m.URI,
			"method": m.Method,
			"error":  err,
		}).Errorf("Could not process the request correctly. data: %s", m.Data)
		response = simpleResponse(400)
		err = nil
	}

	logrus.WithFields(logrus.Fields{
		"response": response,
		"err":      err,
	}).Debugf("Sending response. method: %s, uri: %s", m.Method, m.URI)

	return response, err
}

// getFilters parses the query and returns filters
func getFilters(u *url.URL) map[string]string {
	res := map[string]string{}

	keys := make([]string, 0, len(u.Query()))
	for k := range u.Query() {
		keys = append(keys, k)
	}

	for _, k := range keys {
		if strings.HasPrefix(k, "filter_") {
			tmp, _ := strings.CutPrefix(k, "filter_")
			res[tmp] = u.Query().Get(k)
		}
	}

	return res
}

// convertToChatFilters converts string filters to typed chat filters
func convertToChatFilters(strFilters map[string]string) map[chat.Field]any {
	res := make(map[chat.Field]any)
	for k, v := range strFilters {
		switch k {
		case "customer_id":
			res[chat.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "type":
			res[chat.FieldType] = chat.Type(v)
		case "room_owner_id":
			res[chat.FieldRoomOwnerID] = uuid.FromStringOrNil(v)
		case "participant_ids":
			res[chat.FieldParticipantIDs] = v // string for participant_ids filter
		case "deleted":
			res[chat.FieldDeleted] = v == "true"
		default:
			// skip unknown filters
		}
	}
	return res
}

// convertToChatroomFilters converts string filters to typed chatroom filters
func convertToChatroomFilters(strFilters map[string]string) map[chatroom.Field]any {
	res := make(map[chatroom.Field]any)
	for k, v := range strFilters {
		switch k {
		case "customer_id":
			res[chatroom.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "owner_id":
			res[chatroom.FieldOwnerID] = uuid.FromStringOrNil(v)
		case "chat_id":
			res[chatroom.FieldChatID] = uuid.FromStringOrNil(v)
		case "type":
			res[chatroom.FieldType] = chatroom.Type(v)
		case "deleted":
			res[chatroom.FieldDeleted] = v == "true"
		default:
			// skip unknown filters
		}
	}
	return res
}

// convertToMessagechatFilters converts string filters to typed messagechat filters
func convertToMessagechatFilters(strFilters map[string]string) map[messagechat.Field]any {
	res := make(map[messagechat.Field]any)
	for k, v := range strFilters {
		switch k {
		case "customer_id":
			res[messagechat.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "chat_id":
			res[messagechat.FieldChatID] = uuid.FromStringOrNil(v)
		case "type":
			res[messagechat.FieldType] = messagechat.Type(v)
		case "deleted":
			res[messagechat.FieldDeleted] = v == "true"
		default:
			// skip unknown filters
		}
	}
	return res
}

// convertToMessagechatroomFilters converts string filters to typed messagechatroom filters
func convertToMessagechatroomFilters(strFilters map[string]string) map[messagechatroom.Field]any {
	res := make(map[messagechatroom.Field]any)
	for k, v := range strFilters {
		switch k {
		case "customer_id":
			res[messagechatroom.FieldCustomerID] = uuid.FromStringOrNil(v)
		case "chatroom_id":
			res[messagechatroom.FieldChatroomID] = uuid.FromStringOrNil(v)
		case "owner_id":
			res[messagechatroom.FieldOwnerID] = uuid.FromStringOrNil(v)
		case "messagechat_id":
			res[messagechatroom.FieldMessagechatID] = uuid.FromStringOrNil(v)
		case "type":
			res[messagechatroom.FieldType] = messagechatroom.Type(v)
		case "deleted":
			res[messagechatroom.FieldDeleted] = v == "true"
		default:
			// skip unknown filters
		}
	}
	return res
}

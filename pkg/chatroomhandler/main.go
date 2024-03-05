package chatroomhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package chatroomhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
	"gitlab.com/voipbin/bin-manager/chat-manager.git/pkg/dbhandler"
)

// chatroomHandler defines
type chatroomHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler
}

// ChatroomHandler defines
type ChatroomHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error)
	Gets(ctx context.Context, token string, limit uint64, filters map[string]string) ([]*chatroom.Chatroom, error)
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		agentID uuid.UUID,
		chatroomType chatroom.Type,
		chatID uuid.UUID,
		ownerID uuid.UUID,
		participantIDs []uuid.UUID,
		name string,
		detail string,
	) (*chatroom.Chatroom, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*chatroom.Chatroom, error)
	AddParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatroom.Chatroom, error)
	RemoveParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chatroom.Chatroom, error)
	Delete(ctx context.Context, id uuid.UUID) (*chatroom.Chatroom, error)
}

// NewChatroomHandler returns new ChatroomHandler
func NewChatroomHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,
) ChatroomHandler {

	return &chatroomHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,
	}
}

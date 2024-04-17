package chathandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package chathandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/chat"
	"monorepo/bin-chat-manager/pkg/chatroomhandler"
	"monorepo/bin-chat-manager/pkg/dbhandler"
)

// chatHandler defines
type chatHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	reqHandler    requesthandler.RequestHandler
	notifyHandler notifyhandler.NotifyHandler

	chatroomHandler chatroomhandler.ChatroomHandler
}

// ChatHandler defines
type ChatHandler interface {
	Get(ctx context.Context, id uuid.UUID) (*chat.Chat, error)
	Gets(ctx context.Context, token string, limit uint64, filters map[string]string) ([]*chat.Chat, error)
	Create(
		ctx context.Context,
		customerID uuid.UUID,
		chatType chat.Type,
		ownerID uuid.UUID,
		participantIDs []uuid.UUID,
		name string,
		detail string,
	) (*chat.Chat, error)
	UpdateBasicInfo(ctx context.Context, id uuid.UUID, name string, detail string) (*chat.Chat, error)
	UpdateOwnerID(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*chat.Chat, error)
	AddParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chat.Chat, error)
	RemoveParticipantID(ctx context.Context, id uuid.UUID, participantID uuid.UUID) (*chat.Chat, error)
	Delete(ctx context.Context, id uuid.UUID) (*chat.Chat, error)
}

// NewChatHandler returns new ChatHandler
func NewChatHandler(
	db dbhandler.DBHandler,
	reqHandler requesthandler.RequestHandler,
	notifyHandler notifyhandler.NotifyHandler,

	chatroomHandler chatroomhandler.ChatroomHandler,
) ChatHandler {

	return &chatHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		reqHandler:    reqHandler,
		notifyHandler: notifyHandler,

		chatroomHandler: chatroomHandler,
	}
}

// // sortParticipantIDs sort the given participant ids
// func sortParticipantIDs(participantIDs []uuid.UUID) {
// 	// sort the participants
// 	sort.Slice(participantIDs, func(i, j int) bool {
// 		return participantIDs[i].String() < participantIDs[j].String()
// 	})
// }

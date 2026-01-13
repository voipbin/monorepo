package chathandler

//go:generate mockgen -package chathandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"sort"
	"strings"

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
	Gets(ctx context.Context, token string, limit uint64, filters map[chat.Field]any) ([]*chat.Chat, error)
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
	UpdateRoomOwnerID(ctx context.Context, id uuid.UUID, ownerID uuid.UUID) (*chat.Chat, error)
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

// sortByUUID implements sort.Interface for []uuid.UUID based on the string representation of the UUIDs.
type sortByUUID []uuid.UUID

func (a sortByUUID) Len() int           { return len(a) }
func (a sortByUUID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sortByUUID) Less(i, j int) bool { return a[i].String() < a[j].String() }

// sortUUIDs sorts the given list of UUIDs.
func sortUUIDs(uuids []uuid.UUID) []uuid.UUID {
	res := make([]uuid.UUID, len(uuids))
	copy(res, uuids)

	sort.Sort(sortByUUID(res))

	return res
}

func convertUUIDsToCommaSeparatedString(uuids []uuid.UUID) string {
	strUUIDs := make([]string, len(uuids))
	for i, u := range uuids {
		strUUIDs[i] = u.String()
	}
	return strings.Join(strUUIDs, ",")
}

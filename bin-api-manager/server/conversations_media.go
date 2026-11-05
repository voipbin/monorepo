package server

import (
	"monorepo/bin-api-manager/gens/openapi_server"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cvmedia "monorepo/bin-conversation-manager/models/media"

	"github.com/gofrs/uuid"
)

// convertConversationMedias converts the medias of a conversation message request to the
// conversation-manager medias.
//
// Only id, type and filename are taken from the request, so a client cannot set the customer id or
// the timestamps. For a conversation that sends an sms, the id is the storage file id of the file to
// attach. A missing or malformed id becomes uuid.Nil. This function does not reject anything: the
// conversation-manager rejects a media without a usable id when the conversation can send it. The
// result is never nil.
func convertConversationMedias(in []openapi_server.ConversationManagerMedia) []cvmedia.Media {
	res := make([]cvmedia.Media, 0, len(in))
	for _, m := range in {
		tmp := cvmedia.Media{}

		var id uuid.UUID
		if m.Id != nil {
			id = uuid.FromStringOrNil(*m.Id)
		}
		tmp.Identity = commonidentity.Identity{ID: id}

		if m.Type != nil {
			tmp.Type = cvmedia.Type(*m.Type)
		}
		if m.Filename != nil {
			tmp.Filename = *m.Filename
		}

		res = append(res, tmp)
	}

	return res
}

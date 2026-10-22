package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	tkchat "monorepo/bin-talk-manager/models/chat"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkparticipant "monorepo/bin-talk-manager/models/participant"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const (
	s1TalkChatID    = "873b5e00-0014-11f0-9144-6b8cd8e94f14"
	s1TalkMsgID     = "984c6f11-0015-11f0-8255-7c9de9fa5025"
	s1TalkPartID    = "a95d7022-0016-11f0-9366-8daefa0b6136"
	s1TalkParentID  = "ba6e8133-0017-11f0-8477-9ebf0b1c7247"
	s1TalkMediaFile = "cb7f9244-0018-11f0-9588-afc01c2d8358"
)

func Test_GetServiceAgentsTalkChats_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/service_agents/talk_chats", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().ServiceAgentTalkChatList(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_GetServiceAgentsTalkChannels_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/service_agents/talk_channels", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().ServiceAgentTalkChannelList(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_PostServiceAgentsTalkChats_Errors(t *testing.T) {
	path := "/service_agents/talk_chats"
	body := `{"type":"group","name":"n","detail":"d","participants":[{"owner_type":"agent","owner_id":"` + s1AgentID + `"}]}`
	s1Run(t, []s1Case{
		s1Unauth("POST", path, body),
		s1BadJSON("POST", path),
		s1BadID("POST", path, `{"type":"group","participants":[{"owner_type":"agent","owner_id":"not-a-uuid"}]}`),
		s1PermDenied("POST", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkChatCreate(gomock.Any(), a, tkchat.Type("group"), "n", "d",
				[]tkparticipant.ParticipantInput{{OwnerType: "agent", OwnerID: s1ID(s1AgentID)}}).Return(nil, err)
		}),
		s1PermDenied("POST", path, `{"type":"direct"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkChatCreate(gomock.Any(), a, tkchat.Type("direct"), "", "", []tkparticipant.ParticipantInput(nil)).Return(nil, err)
		}),
	})
}

func Test_ServiceAgentsTalkChatsId_Errors(t *testing.T) {
	path := "/service_agents/talk_chats/" + s1TalkChatID
	bad := "/service_agents/talk_chats/not-a-uuid"
	id := s1ID(s1TalkChatID)
	putBody := `{"name":"n2","detail":"d2"}`
	name, detail := "n2", "d2"
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", bad, ""),
		s1NotFound("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkChatGet(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("PUT", path, putBody),
		s1BadID("PUT", bad, putBody),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, putBody, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkChatUpdate(gomock.Any(), a, id, &name, &detail).Return(nil, err)
		}),
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", bad, ""),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkChatDelete(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("POST", path+"/join", ""),
		s1BadID("POST", bad+"/join", ""),
		s1PermDenied("POST", path+"/join", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkChatJoin(gomock.Any(), a, id).Return(nil, err)
		}),
	})
}

func Test_ServiceAgentsTalkChatsIdParticipants_Errors(t *testing.T) {
	path := "/service_agents/talk_chats/" + s1TalkChatID + "/participants"
	bad := "/service_agents/talk_chats/not-a-uuid/participants"
	id := s1ID(s1TalkChatID)
	postBody := `{"owner_type":"agent","owner_id":"` + s1AgentID + `"}`
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", bad, ""),
		s1PermDenied("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkParticipantList(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("POST", path, postBody),
		s1BadID("POST", bad, postBody),
		s1BadJSON("POST", path),
		s1BadID("POST", path, `{"owner_type":"agent","owner_id":"not-a-uuid"}`),
		s1PermDenied("POST", path, postBody, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkParticipantCreate(gomock.Any(), a, id, "agent", s1ID(s1AgentID)).Return(nil, err)
		}),
	})
}

func Test_DeleteServiceAgentsTalkChatsIdParticipantsParticipantId_Errors(t *testing.T) {
	path := "/service_agents/talk_chats/" + s1TalkChatID + "/participants/" + s1TalkPartID
	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/service_agents/talk_chats/not-a-uuid/participants/"+s1TalkPartID, ""),
		s1BadID("DELETE", "/service_agents/talk_chats/"+s1TalkChatID+"/participants/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkParticipantDelete(gomock.Any(), a, s1ID(s1TalkChatID), s1ID(s1TalkPartID)).Return(nil, err)
		}),
	})
}

func Test_GetServiceAgentsTalkMessages_Errors(t *testing.T) {
	base := "/service_agents/talk_messages"
	chatQuery := "?chat_id=" + s1TalkChatID
	s1Run(t, []s1Case{
		s1Unauth("GET", base+chatQuery, ""),
		s1BadID("GET", base+"?chat_id=not-a-uuid", ""),
		s1PermDenied("GET", base+chatQuery+"&page_size=500&page_token=tok", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkMessageList(gomock.Any(), a, s1ID(s1TalkChatID), uint64(100), "tok").Return(nil, err)
		}),
		{
			name:   "empty result",
			method: "GET",
			path:   base + chatQuery,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentTalkMessageList(gomock.Any(), a, s1ID(s1TalkChatID), uint64(100), "").Return(nil, nil)
			},
			wantBody: `{"result":[],"next_page_token":""}`,
		},
	})
}

func Test_PostServiceAgentsTalkMessages_Errors(t *testing.T) {
	path := "/service_agents/talk_messages"
	body := `{"chat_id":"` + s1TalkChatID + `","parent_id":"` + s1TalkParentID + `","type":"normal","text":"hi","medias":[{"type":"link","link_url":"https://example.com/a"},{"type":"file","file_id":"` + s1TalkMediaFile + `"},{}]}`
	parent := s1ID(s1TalkParentID)
	s1Run(t, []s1Case{
		s1Unauth("POST", path, body),
		s1BadJSON("POST", path),
		s1BadID("POST", path, `{"chat_id":"not-a-uuid","type":"normal","text":"hi"}`),
		s1PermDenied("POST", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkMessageCreate(gomock.Any(), a, s1ID(s1TalkChatID), &parent, tkmessage.Type("normal"), "hi", []tkmessage.Media{
				{Type: tkmessage.MediaType("link"), LinkURL: "https://example.com/a"},
				{Type: tkmessage.MediaType("file"), FileID: s1ID(s1TalkMediaFile)},
				{},
			}).Return(nil, err)
		}),
		s1PermDenied("POST", path, `{"chat_id":"`+s1TalkChatID+`","parent_id":"`+s1NilUUID+`","type":"normal","text":"hi"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkMessageCreate(gomock.Any(), a, s1ID(s1TalkChatID), (*uuid.UUID)(nil), tkmessage.Type("normal"), "hi", []tkmessage.Media(nil)).Return(nil, err)
		}),
	})
}

func Test_ServiceAgentsTalkMessagesId_Errors(t *testing.T) {
	path := "/service_agents/talk_messages/" + s1TalkMsgID
	bad := "/service_agents/talk_messages/not-a-uuid"
	id := s1ID(s1TalkMsgID)
	reactBody := `{"emoji":"👍"}`
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", bad, ""),
		s1NotFound("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkMessageGet(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", bad, ""),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkMessageDelete(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("POST", path+"/reactions", reactBody),
		s1BadID("POST", bad+"/reactions", reactBody),
		s1BadJSON("POST", path+"/reactions"),
		s1PermDenied("POST", path+"/reactions", reactBody, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkMessageReactionCreate(gomock.Any(), a, id, "👍").Return(nil, err)
		}),
	})
}

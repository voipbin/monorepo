import { createStore, applyMiddleware, combineReducers } from 'redux'

const SET = 'set';

const initialState = {
  sidebarShow: true,
  sidebarUnfoldable: false,
}

///////////////////////////////////////////////////////////
// resource chatroommessages
const resourceChatroommessagesInitial = {
  data: {},
}

export const CHATROOMMESSAGES_SET_WITH_CHATROOMID = 'CHATROOMMESSAGES_SET_WITH_ROOMID';
export const ChatroommessagesSetWithChatroomID = (chatroomID, data) => {
  return {
    type: CHATROOMMESSAGES_SET_WITH_CHATROOMID,
    chatroom_id: chatroomID,
    data: data,
  }
}

export const CHATROOMMESSAGES_ADD_WITH_CHATROOMID = 'CHATROOMMESSAGES_ADD_WITH_CHATROOMID';
export const ChatroommessagesAddWithChatroomID = (chatroomID, data) => {
  return {
    type: CHATROOMMESSAGES_ADD_WITH_CHATROOMID,
    chatroom_id: chatroomID,
    data: data,
  }
}

const resourceChatroommessagesReducer = (state = resourceChatroommessagesInitial, action) => {
  let newState = {
    data: state.data,
  };

  switch (action.type) {
    case CHATROOMMESSAGES_SET_WITH_CHATROOMID:
      newState.data[action.chatroom_id] = action.data;
      return newState;

    case CHATROOMMESSAGES_ADD_WITH_CHATROOMID:
      const roomID = action.data["chatroom_id"];
      if (newState.data[roomID]) {
        newState.data[roomID] = [action.data, ...newState.data[roomID]];
      } else {
        newState.data[roomID] = [action.data];
      }
      return newState;

    default:
      return newState;
  }
}


///////////////////////////////////////////////////////////
// resource transcripts
const resourceTranscriptsInitial = {
  data: {},
}

export const TRANSCRIPTS_SET_WITH_CHATROOMID = 'TRANSCRIPTS_SET_WITH_TRANSCRIBEID';
export const TranscriptsSetWithTranscribeID = (transcribeID, data) => {
  return {
    type: CHATROOMMESSAGES_SET_WITH_CHATROOMID,
    chatroom_id: transcribeID,
    data: data,
  }
}

export const TRANSCRIPTS_ADD_WITH_TRANSCRIBEID = 'TRANSCRIPTS_ADD_WITH_TRANSCRIBEID';
export const TranscriptsAddWithTranscribeID = (transcribeID, data) => {
  return {
    type: TRANSCRIPTS_ADD_WITH_TRANSCRIBEID,
    chatroom_id: transcribeID,
    data: data,
  }
}

const resourceTranscriptsReducer = (state = resourceTranscriptsInitial, action) => {
  let newState = {
    data: state.data,
  };

  switch (action.type) {
    case TRANSCRIPTS_SET_WITH_CHATROOMID:
      newState.data[action.chatroom_id] = action.data;
      return newState;

    case TRANSCRIPTS_ADD_WITH_TRANSCRIBEID:
      const roomID = action.data["transcribe_id"];
      if (newState.data[roomID]) {
        newState.data[roomID] = [action.data, ...newState.data[roomID]];
      } else {
        newState.data[roomID] = [action.data];
      }
      return newState;

    default:
      return newState;
  }
}


///////////////////////////////////////////////////////////
// resource currentcall
const resourceCurrentcallInitial = {
  "session": {},
  "id": "",
  "source": "",
  "destination": "",
  "status": "finished",
  "direction": "",
}

export const CURRENTCALL_SET_SESSION = 'CURRENTCALL_SET_SESSION';
export const CurrentcallSetSession = (session) => {
  return {
    type: CURRENTCALL_SET_SESSION,
    data: session,
  }
}

export const CURRENTCALL_SET_STATUS = 'CURRENTCALL_SET_STATUS';
export const CurrentcallSetStatus = (state) => {
  return {
    type: CURRENTCALL_SET_STATUS,
    data: state,
  }
}

const resourceCurrentcallReducer = (state = resourceCurrentcallInitial, action) => {
  let newState = state;

  switch (action.type) {
    case CURRENTCALL_SET_SESSION:
      console.log("Setting the currentcall session. action: ", action)
      newState["session"] = action.data;
      newState["id"] = action.data.id;
      newState["direction"] = action.data.direction;

      if (action.data.remote_identity == null) {
        return newState;
      }

      if (newState['direction'] == 'incoming') {
        // incoming
        newState['source'] = action.data.remote_identity.uri.user;
        newState['destination'] = action.data.local_identity.uri.user;

        newState['status'] = 'ringing';
      } else {
        // outgoing
        newState['source'] = action.data.local_identity.uri.user;
        newState['destination'] = action.data.remote_identity.uri.user;

        newState['status'] = 'dialing';
      }

      return newState;

    case CURRENTCALL_SET_STATUS:
      console.log("Setting the current call status. action: ", action)
      newState["status"] = action.data;
      return newState;

    default:
      return newState;
  }
}

///////////////////////////////////////////////////////////
// resource agent info
const resourceAgentInfoInitial = {
  "data": {},
}

export const AGENTINFO_SET = 'AGENTINFO_SET';
export const AgentInfoSet = (data) => {
  return {
    type: AGENTINFO_SET,
    data: data,
  }
}

const resourceAgentInfoReducer = (state = resourceAgentInfoInitial, action) => {
  let newState = state;

  switch (action.type) {
    case AGENTINFO_SET:
      console.log("Setting the agent info data. action: ", action)
      newState["data"] = action.data;
      return newState;

    default:
      return newState;
  }
}

///////////////////////////////////////////////////////////
// 
const resourceReducer = (state = initialState, action) => {

  switch (action.type) {
    case 'update':
      console.log("Update data. ", action.type, action.data);
      localStorage.setItem(action.type, JSON.stringify(action.data));

    default:
      return state;
  }
}

///////////////////////////////////////////////////////////
const storeAll = combineReducers({
  resourceReducer,
  resourceChatroommessagesReducer,
  resourceCurrentcallReducer,
  resourceAgentReducer: resourceAgentInfoReducer,
});

const store = createStore(storeAll);
export default store

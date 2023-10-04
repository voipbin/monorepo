import { createStore, applyMiddleware } from 'redux'
import myLogger from './middleware';

const SET = 'set';

const initialState = {
  sidebarShow: true,
  sidebarUnfoldable: false,
}

const changeState = (state = initialState, action) => {

  switch (action.method) {
    case 'update':
      console.log("Update data. ", action.type, action.data);
      localStorage.setItem(action.type, JSON.stringify(action.data));

    // case 'get':
    //   return localStorage.getItem(action.type)
  }

  switch (action.type) {
    case 'activeflows':
      return Object.assign({}, state, { ...state, activeflows: action.data });
    case 'agents':
      return Object.assign({}, state, { ...state, agents: action.data });
    case 'billing_accounts':
      return Object.assign({}, state, { ...state, billing_accounts: action.data });
    case 'calls':
      return Object.assign({}, state, { ...state, calls: action.data });
    case 'campaigns':
      return Object.assign({}, state, { ...state, campaigns: action.data });
    case 'chatbots':
      return Object.assign({}, state, { ...state, chatbots: action.data });
    case 'chats':
      return Object.assign({}, state, { ...state, chats: action.data });
    case 'customers':
      return Object.assign({}, state, { ...state, customers: action.data });
    case 'flows':
      return Object.assign({}, state, { ...state, flows: action.data });
    case 'groupcalls':
      return Object.assign({}, state, { ...state, groupcalls: action.data });
    case 'messages':
      return Object.assign({}, state, { ...state, messages: action.data });
    case 'numbers':
      return Object.assign({}, state, { ...state, numbers: action.data });
    case 'queues':
      return Object.assign({}, state, { ...state, queues: action.data });
    case 'queuecalls':
      return Object.assign({}, state, { ...state, queuecalls: action.data });
    default:
      return state
  }
}

const store = createStore(changeState);
export default store

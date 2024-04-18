import { useRef, useState, useCallback, useEffect } from 'react'
import { useParams, useNavigate, useLocation } from "react-router-dom";
import 'reactflow/dist/style.css';
import "./styles.css";
import React from 'react';
import ReactFlow, { 
  ReactFlowProvider, 
  Background, 
  Panel, 
  Controls,
  MiniMap,
  useReactFlow,
 } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { useStore } from './store.js';
import { tw } from 'twind';
import {
  CButton,
} from '@coreui/react'
import {
  resetActions,
  getActions,
} from './action.js';
import SideBar from './sidebar.js';

import Start from './nodes/start.js';
import AMD from './nodes/amd.js';
import Answer from './nodes/answer.js';
import Beep from './nodes/beep.js';
import Branch from './nodes/branch.js';
import Call from './nodes/call.js';
import ChatbotTalk from './nodes/chatbot_talk.js';
import ConfbridgeJoin from './nodes/confbridge_join.js';
import ConferenceJoin from './nodes/conference_join.js';
import Connect from './nodes/connect.js';
import ConversationSend from './nodes/conversation_send.js';
import DigitsReceive from './nodes/digits_receive.js';
import DigitsSend from './nodes/digits_send.js';
import Echo from './nodes/echo.js';
import ExternalMediaStart from './nodes/external_media_start.js';
import ExternalMediaStop from './nodes/external_media_stop.js';
import Fetch from './nodes/fetch.js';
import FetchFlow from './nodes/fetch_flow.js';
import Goto from './nodes/goto.js';
import Hangup from './nodes/hangup.js';
import MessageSend from './nodes/message_send.js';
import Play from './nodes/play.js';
import QueueJoin from './nodes/queue_join.js';
import RecordingStart from './nodes/recording_start.js';
import RecordingStop from './nodes/recording_stop.js';
import Sleep from './nodes/sleep.js';
import Stop from './nodes/stop.js';
import StreamEcho from './nodes/stream_echo.js';
import Talk from './nodes/talk.js';
import TranscribeStart from './nodes/transcribe_start.js';
import TranscribeStop from './nodes/transcribe_stop.js';
import TranscribeRecording from './nodes/transcribe_recording.js';
import VariableSet from './nodes/variable_set.js';
import WebhookSend from './nodes/webhook_send.js';




import 'reactflow/dist/style.css';

const nodeTypes = {
  start: Start,
  amd: AMD,
  answer: Answer,
  beep: Beep,
  branch: Branch,
  call: Call,
  chatbot_talk: ChatbotTalk,
  confbridge_join: ConfbridgeJoin,
  conference_join: ConferenceJoin,
  connect: Connect,
  conversation_send: ConversationSend,
  digits_receive: DigitsReceive,
  digits_send: DigitsSend,
  echo: Echo,
  external_media_start: ExternalMediaStart,
  external_media_stop: ExternalMediaStop,
  fetch: Fetch,
  fetch_flow: FetchFlow,
  goto: Goto,
  hangup: Hangup,
  message_send: MessageSend,
  play: Play,
  queue_join: QueueJoin,
  recording_start: RecordingStart,
  recording_stop: RecordingStop,
  sleep: Sleep,
  stop: Stop,
  stream_echo: StreamEcho,
  talk: Talk,
  transcribe_start: TranscribeStart,
  transcribe_stop: TranscribeStop,
  transcribe_recording: TranscribeRecording,
  variable_set: VariableSet,
  webhook_send: WebhookSend,
};

const selector = (store) => ({
  nodes: store.nodes,
  edges: store.edges,
  initNodes: store.initNodes,
  initEdges: store.initEdges,

  onNodesChange: store.onNodesChange,
  onNodesDelete: store.onNodesDelete,
  onEdgesChange: store.onEdgesChange,
  onEdgesDelete: store.onEdgesDelete,
  onConnect: store.onConnect,
  onDragOver: store.onDragOver,
  onDrop: store.onDrop,

  onInit: store.onInit,

  createNode: (type) => {
    store.createNode(type);
  },

});

const minimapStyle = {
  height: 120,
  left: 40,
};

const defaultEdgeOptions = {
  animated: false,
  type: 'smoothstep',
};

const defaultProOptions = { 
  hideAttribution: true,
};


export default function ActionGraph() {
  const store = useStore(selector, shallow);
  const navigate = useNavigate();
  const [reactFlowInstance, setReactFlowInstance] = useState(null);

  // parse the params
  const data = useLocation();
  console.log("Debug. uselocation: %o", data);

  const referer = data.state.referer;
  const target = data.state.target;
  const actions = data.state.actions;
  console.log("Parsed data. target: %o, actions: %o", target, actions);

  useEffect(() => {
    InitActions();
    return;
  },[]);

  const navigateBack = (data) => {
    navigate(referer, data);
  }

  const Save = () => {
    const tmp = getActions();
    console.log("Parsed actions. actions: %o", tmp);

    const state = {
      state: {
        "target": target,
        "actions": tmp,
      }
    };
    console.log("Navigating back to target. state: %o", state);
    navigateBack(state);
  }

  const Cancel = () => {
    navigate(-1);
  }

  const InitActions = () => {
    resetActions();
  
    store.initNodes(actions);
    store.initEdges(actions);
  }

  return (
    <>
      <ReactFlowProvider>
        <div style={{  height: '85vh' }}>
          <ReactFlow
            nodeTypes={nodeTypes}
            nodes={store.nodes}
            edges={store.edges}

            onNodesChange={store.onNodesChange}
            onNodesDelete={store.onNodesDelete}
            onEdgesChange={store.onEdgesChange}
            onEdgesDelete={store.onEdgesDelete}
            onConnect={store.onConnect}
            onDragOver={store.onDragOver}
            onDrop={store.onDrop}
            onInit={store.onInit}

            defaultEdgeOptions={defaultEdgeOptions}
            proOptions={defaultProOptions}
            fitView
          >

            <Panel className={tw('space-x-4')} position="top">
              <CButton type="submit" onClick={() => Save()}>Save</CButton>
              <CButton type="submit" color="dark" onClick={() => Cancel()}>Cancel</CButton>
            </Panel>

            <Panel position="top-right">
              <SideBar></SideBar>
            </Panel>
            
            <Controls />
            <MiniMap style={minimapStyle} zoomable pannable position="bottom-left" />
            <Background />
          </ReactFlow>
        </div>
      </ReactFlowProvider>
    </>
  );
}

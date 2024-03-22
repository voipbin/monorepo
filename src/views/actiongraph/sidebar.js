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
  CCard,
  CCardBody,
  CCardHeader,
  CCol,
  CFormInput,
  CFormLabel,
  CRow,
  CFormTextarea,
  CButton,
  CFormSelect,
  } from '@coreui/react'


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

  createNode: (type) => {
    store.createNode(type);
  },

});

export default function SideBar() {
  const store = useStore(selector, shallow);

  const actions = [
    {
      name: 'AMD',
      description: 'Answer Machine Detection',
      type: 'amd',
    },
    {
      name: 'Answer',
      description: 'Answer the call',
      type: 'answer',
    },
    {
      name: 'Beep',
      description: 'Play the beep sound',
      type: 'beep',
    },
    {
      name: 'Branch',
      description: 'Branch gets the variable then execute the correspond action',
      type: 'branch',
    },
    {
      name: 'Call',
      description: 'Starts a new independent outgoing call with a given flow',
      type: 'call',
    },
    {
      name: 'Chatbot Talk(AI)',
      description: 'Starts a talk with AI chatbot',
      type: 'chatbot_talk',
    },
    {
      name: 'Confbridge Join',
      description: 'Join to the confbridge',
      type: 'confbridge_join',
    },
    {
      name: 'Conference Join',
      description: 'Join to the conference',
      type: 'conference_join',
    },
    {
      name: 'Connect',
      description: 'Creates a new call to the destinations and connects to them',
      type: 'connect',
    },
    {
      name: 'Conversation Send',
      description: 'Send a message to the conversation',
      type: 'conversation_send',
    },
    {
      name: 'Digits Receive',
      description: 'Receive the digits(dtmfs)',
      type: 'digits_receive',
    },
    {
      name: 'Digits Send',
      description: 'Send the digits(dtmfs)',
      type: 'digits_send',
    },
    {
      name: 'Echo',
      description: 'Echo to stream',
      type: 'echo',
    },
    {
      name: 'External Media Start',
      description: 'Start the external media',
      type: 'external_media_start',
    },
    {
      name: 'External Media Stop',
      description: 'Stop the external media',
      type: 'external_media_stop',
    },
    {
      name: 'Fetch',
      description: 'Fetch the actions from endpoint',
      type: 'fetch',
    },
    {
      name: 'Fetch Flow',
      description: 'Fetch the actions from the exist flow',
      type: 'fetch_flow',
    },
    {
      name: 'Goto',
      description: 'Move the next action cursor to the given action',
      type: 'goto',
    },
    {
      name: 'Hangup',
      description: 'Hangup the call',
      type: 'hangup',
    },
    {
      name: 'Message Send',
      description: 'Send a message',
      type: 'message_send',
    },
    {
      name: 'Play',
      description: 'Play the file of the given urls',
      type: 'play',
    },
    {
      name: 'Queue Join',
      description: 'Join to the queue',
      type: 'queue_join',
    },
    {
      name: 'Recording Start',
      description: 'Start the record of the given call',
      type: 'recording_start',
    },
    {
      name: 'Recording Stop',
      description: 'Stop the record of the given call',
      type: 'recording_stop',
    },
    {
      name: 'Sleep',
      description: 'Sleep',
      type: 'sleep',
    },
    {
      name: 'Stop',
      description: 'Stop the flow',
      type: 'stop',
    },
    {
      name: 'Stream Echo',
      description: 'Echo the steam',
      type: 'stream_echo',
    },
    {
      name: 'Talk',
      description: 'Generate audio from the given text(ssml or plain text) and play it.',
      type: 'talk',
    },
    {
      name: 'Transcribe Start',
      description: 'Start transcribe the call',
      type: 'transcribe_start',
    },
    {
      name: 'Transcribe Stop',
      description: 'Stop transcribe the call',
      type: 'transcribe_stop',
    },
    {
      name: 'Transcribe Recording',
      description: 'Transcribe the recording and send it to webhook',
      type: 'transcribe_recording',
    },
    {
      name: 'Variable Set',
      description: 'Sets the variable',
      type: 'variable_set',
    },
    {
      name: 'Webhook Send',
      description: 'Send a webhook',
      type: 'webhook_send',
    },    
  ];
  
  const onDragStart = (event, nodeType) => {
    console.log("onDragStart. nodeType: %o, event: %o", nodeType, event);
    event.dataTransfer.setData("application/reactflow", nodeType);
    event.dataTransfer.effectAllowed = "move";
  };

  return(
    <div class="list-group" style={{overflowY:'auto',height:'83vh',width:'10vw',cursor:'pointer'}}>
      {actions.map(function(action, i){
        return (
          <div className="dndnode" draggable onDragStart={(event) => onDragStart(event, action.type)}>
            <a onClick={() => store.createNode(action.type)} class="list-group-item list-group-item-action list-group-item-dark" aria-current="true">
              <div class="d-flex w-100 justify-content-between">
                <b class="mb-1">{action.name}</b>
              </div>
              <small class="mb-1">{action.description}</small>
            </a>
          </div>
        )
      })}
    </div>
  );
}
import { applyNodeChanges, applyEdgeChanges } from 'reactflow';
import { nanoid } from 'nanoid';
import { create } from 'zustand';
import { createStore } from 'zustand'
import { useRef, useState, useCallback, useEffect } from 'react'


import {
  addAction,
  updateAction,
  removeAction,
  connectAction,
  disconnectAction,
} from './action';
import { v4 as uuidv4 } from 'uuid';

const startID = "00000000-0000-0000-0000-000000000001";
const emptyID = "00000000-0000-0000-0000-000000000000";

export const useStore = create((set, get) => ({
  instance: null,
  // instance: [reactFlowInstance, setReactFlowInstance] = useState(null),
  // reactFlowInstance, setReactFlowInstance: useState(null),

  nodes: [
    { id: startID, type: 'start',  deletable: false, position: { x: 0, y: 0 } }
  ],
  edges: [],

  onNodesChange(changes) {
    console.log("onNodesChange. changes: %o", changes);
    set({
      nodes: applyNodeChanges(changes, get().nodes),
    });
  },

  onInit(instance) {
    console.log("onInit. %o", instance);
    if (instance != null) {
      console.log("onInit. instance: %o", instance);
      set({
        instance: instance,
      })

    }
  },

  createNode(type, x, y) {
    const id = uuidv4();

    const position = { x: x, y: y };

    var data = {};
    switch (type) {
      case 'amd': {
        data = {
          machine_handle: 'hangup',
          sync: true,
        }
        break;
      }

      case 'answer': {
        data = {};
        break;
      }

      case 'beep': {
        data = {};
        break;
      }

      case 'branch': {
        data = {
          variable: 'voipbin.call.digits',
          default_target_id: '',
          target_ids: {},
        };
        break;
      }

      case 'call': {
        data = {
          source: {
            "type": "tel",
            "target": ""
          },
          destinations: [
            {
              "type": "tel",
              "target": ""
            }
          ],
          flow_id: emptyID,
          actions: [],
          chained: false,
          early_execution: false,
        };
        break;
      }

      case 'chatbot_talk': {
        data = {
          chatbot_id: emptyID,
          gender: "male",
          language: "en-US",
          duration: 3600
        };
        break;
      }

      case 'confbridge_join': {
        data = {
          confbridge_id: emptyID,
        };
        break;
      }

      case 'conference_join': {
        data = {
          conference_id: emptyID,
        };
        break;
      }

      case 'connect': {
        data = {
          source: {
            "type": "tel",
            "target": ""
          },
          destinations: [
            {
              "type": "tel",
              "target": ""
            }
          ],
          early_media: false,
          relay_reason: false,
        };
        break;
      }

      case 'conversation_send': {
        data = {
          conversation_id: emptyID,
          text: "hello world",
          sync: false,
        };
        break;
      }

      case 'digits_receive': {
        data = {
          duration: 10000,
          key: "#",
          length: 0,
        };
        break;
      }

      case 'digits_send': {
        data = {
          digits: "123",
          duration: 1000,
          interval: 500,
        };
        break;
      }

      case 'echo': {
        data = {
          duration: 10000,
        };
        break;
      }

      case 'external_media_start': {
        data = {
          external_host: 'example.com',
          encapsulation: 'rtp',
          transport: 'udp',
          connection_type: 'client',
          format: 'ulaw',
          direction: 'both',
          data: '',
        };
        break;
      }
      
      case 'external_media_stop': {
        data = {};
        break;
      }

      case 'fetch': {
        data = {
          event_url: "example.com",
          event_method: "POST",
        };
        break;
      }

      case 'fetch_flow': {
        data = {
          flow_id: emptyID,
        };
        break;
      }

      case 'goto': {
        data = {
          target_id: emptyID,
          loop_count: 1,
        };
        break;
      }

      case 'hangup': {
        data = {
          reason: "normal",
          reference_id: emptyID,
        };
        break;
      }

      case 'message_send': {
        data = {
          source: {
            "type": "tel",
            "target": ""
          },
          destinations: [
            {
              "type": "tel",
              "target": ""
            }
          ],
          text: "hello world",
        };
        break;
      }

      case 'play': {
        data = {
          stream_urls: [
            "example.com/play/file/url",
          ],
        };
        break;
      }

      case 'queue_join': {
        data = {
          queue_id: emptyID,
        };
        break;
      }

      case 'recording_start': {
        data = {
          format: "wav",
          end_of_silence: 0,
          end_of_key: "",
          duration: 0,
          beep_start: false,
        };
        break;
      }

      case 'recording_stop': {
        data = {};
        break;
      }

      case 'sleep': {
        data = {
          duration: 1000,
        };
        break;
      }

      case 'stop': {
        data = {};
        break;
      }

      case 'stream_echo': {
        data = {
          duration: 10000,
        };
        break;
      }

      case 'talk': {
        data = {
          gender: 'male',
          language: 'en-US',
          digits_handle: '',
          text: 'Hello, world',
        };
        break;
      }

      case 'transcribe_start': {
        data = {
          language: 'en-US',
        };
        break;
      }

      case 'transcribe_stop': {
        data = {};
        break;
      }

      case 'transcribe_recording': {
        data = {
          language: 'en-US',
        };
        break;
      }

      case 'variable_set': {
        data = {
          key: "variable key",
          value: "variable value",
        };
        break;
      }

      case 'webhook_send': {
        data = {
          sync: false,
          uri: "example.com/webhook/send/uri",
          method: "POST",
          data_type: "application/json",
          data: "Hello, world",
        };
        break;
      }


      default: {
        console.log("Unsupported type: %s", type);
        return;
      }
    }

    addAction(id, type, emptyID, data);
    set({ nodes: [...get().nodes, { id, type, data, position }] });

    console.log("Print all nodes: ", ...get().nodes);
  },

  initNodes(nodes) {
    console.log("Initiating node. nodes: ", nodes)

    set({nodes: [
      { id: startID, type: 'start',  deletable: false, position: { x: 0, y: 0 } }
    ]});

    set({
      nodes: applyNodeChanges(get().nodes, get().nodes),
    });


    let lastY = 100;
    for (let i = 0; i < nodes.length; i++) {
      const node = nodes[i];
      console.log("initNodes node. node: %o", node);
      node.position = { x: 0, y: lastY };  

      if (node.type == "talk") {
        lastY += 350;
      } else {
        lastY += 200;
      }

      node["data"] = node.option;
      if (node["data"] == undefined || node["data"] == "") {
        node["data"] = {};
        console.log("The node has invalid option. Created a default option. node: %o", node);
      }

      if (node["id"] == undefined ||node["id"] == "" || node["id"] == startID) {
        node["id"] = uuidv4();
        console.log("The node has no id. Created a new id. node: %o", node);
      }
      
      // add nodes
      console.log("Adding a init node. node: %o", node);
      set({nodes: [...get().nodes, node]});

      addAction(node.id, node.type, node.next_id, node.option);
      console.log("initNodes node. node_all: ", ...get().nodes);
    }
  },

  initEdges(nodes) {
    console.log("Initiating edges. nodes: ", nodes)

    // create an edge for start node to the first init node.
    if (nodes.length > 0) {
      const edge = {
        source: startID,
        sourceHandle: null,
        target: nodes[0].id,
        targetHandle: nodes[0].id + "-target",
      }
      console.log("Adding an edge. edge: %o", edge);
      set({ edges: [edge, ...get().edges] });
    }
    
    for (let i = 0; i < nodes.length; i++) {
      if (nodes[i]["type"] === 'branch') {
        if (nodes[i].option.default_target_id != emptyID && nodes[i].option.default_target_id != "") {
          const edge = {
            source: nodes[i].id,
            sourceHandle: nodes[i].id + "-source",
            target: nodes[i].option.default_target_id,
            targetHandle: nodes[i].option.default_target_id + "-target",
          }
          set({ edges: [...get().edges, edge] });          
        }

        const keys = Object.keys(nodes[i].option.target_ids);
        for (let j = 0; j < keys.length; j++) {
          const edge = {
            source: nodes[i].id,
            sourceHandle: nodes[i].id + "-source_" + j,
            target: nodes[i].option.target_ids[keys[j]],
            targetHandle: nodes[i].option.target_ids[keys[j]] + "-target",
          }
          set({ edges: [...get().edges, edge] });          
        }
      } else if (nodes[i]["type"] === "goto") {
        if (nodes[i].option.target_id != emptyID && nodes[i].option.target_id != "") {
          const edge = {
            source: nodes[i].id,
            sourceHandle: nodes[i].id + "-source_target",
            target: nodes[i].option.target_id,
            targetHandle: nodes[i].option.target_id + "-target",
          }
          set({ edges: [...get().edges, edge] });          
        }
      }

      if (nodes[i]["next_id"] == emptyID || nodes[i]["next_id"] == "") {
        // no next node
        console.log("no next node. skipping...");
        continue
      }

      const edge = {
        source: nodes[i].id,
        sourceHandle: nodes[i].id + "-source",
        target: nodes[i]["next_id"],
        targetHandle: nodes[i]["next_id"] + "-target",
      }
      console.log("Adding an edge. edge: %o", edge);
      set({ edges: [...get().edges, edge] });

    }
  },

  updateNode(id, data) {
    set({
      nodes: get().nodes.map((node) =>
        node.id === id ? { ...node, data: Object.assign(node.data, data) } : node
      ),
    });

    updateAction(id, data);
  },

  onNodesDelete(deleted) {
    for (const { id } of deleted) {
      removeAction(id);
    }
  },

  onEdgesChange(changes) {
    set({
      edges: applyEdgeChanges(changes, get().edges),
    });
  },

  onConnect(data) {
    // source: "00000000-0000-0000-0000-000000000001"
    // sourceHandle: null
    // target: "75ec1345-cfe4-4d9a-bf3a-9b3f80723768"
    // targetHandle: "7c0af20d-7f2a-4c42-976b-f487092902f2"
    console.log("onConnect. data: %o", data);

    // remove previous edge
    let tmps = get().edges;
    for (let i = 0; i < tmps.length; i++) {
      const tmp = tmps[i];

      if (tmps[i].sourceHandle == data.sourceHandle) {
        console.log("onConnect. Found existing edge. Deleting it. edge_id: %s", tmps[i].id);
        tmps.splice(i, 1);
        set({edges: [...tmps]});
        break;
      }
    }

    // create edge
    const id = uuidv4();
    const edge = { id, ...data };
    set({ edges: [edge, ...get().edges] });

    // update action's connection
    connectAction(edge.source, edge.sourceHandle, edge.target, edge.targetHandle);
  },

  onDragOver(event) {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  },


  onDrop(event) {
    event.preventDefault();
    
    const type = event.dataTransfer.getData("application/reactflow");
    console.log("onDrop 1111. type: %o, event: %o", type, event);

    // check if the dropped element is valid
    if (typeof type === "undefined" || !type) {
      return;
    }

    // calculate x, y.
    var x = 0;
    var y = 0;

    const instance = get().instance
    if (instance != null) {
      const p = instance.screenToFlowPosition({
          x: event.clientX,
          y: event.clientY,
        });
        console.log("onDrop. p: %o", p);

        x = p.x;
        y = p.y;
    }

    console.log("onDrop. log test. x: %o, y: o", x, y);
    get().createNode(type, x, y);
  },

  onEdgesDelete(deleted) {
    console.log("deleted. deleted: %o", deleted);

    for (let i = 0; i < deleted.length; i++) {
      const tmp = deleted[i];
      disconnectAction(tmp.source);
    }
  },

  onSaveFlow() {
    console.log("Print all node: ", ...get().nodes);
    console.log("Print all edges: ", ...get().edges);
  },
}));

import React, { useRef, useState, useCallback, useEffect } from 'react'
import { useParams } from "react-router-dom";
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
import store from '../../store'
import {
  Get as ProviderGet,
  Post as ProviderPost,
  Put as ProviderPut,
  Delete as ProviderDelete,
  ParseData,
} from '../../provider';
import ReactFlow, {
  applyNodeChanges,
  applyEdgeChanges,
  addEdge,
  Controls,
  Background,
  MiniMap,
  useNodesState,
  useEdgesState,
  ReactFlowProvider,
} from "react-flow-renderer";
import { v4 as uuidv4 } from 'uuid';
import 'reactflow/dist/style.css';
import Sidebar from "./flowgraph_sidebar.js";
import {
  NodeActionTalk,
} from './flowgraph_node.js';
// import OSC from "./nodes/osc.js";
import "./styles.css";
import { tw } from "twind";

// import React from "react";
import { Handle, Position } from "reactflow";
// import { tw } from "twind";
// import { useCallback } from 'react';

// const nodeTypes = { 
//   // nodeTypeActionTalk: NodeActionTalk,
//   osc: OSC,
// };

// const selector = (store) => ({
//   nodes: store.nodes,
//   edges: store.edges,
//   onNodesChange: store.onNodesChange,
//   onEdgesChange: store.onEdgesChange,
//   addEdge: store.addEdge,
// });


const Flowgraph = () => {
  console.log("Flowgraph");


  const nodeTypes = { 
    // nodeTypeActionTalk: NodeActionTalk,
    osc: OSC,
  };
  



  const initialNodes = [
    { id: uuidv4(), type: 'input', data: { label: 'Start' }, position: { x: 250, y: 0 } },
    { id: uuidv4(), type: 'osc', data: { label: 'Start' }, position: { x: 350, y: 0 } },

  ];
  const initialEdges = [];

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);

  const minimapStyle = {
    height: 120,
  };

  const reactFlowWrapper = useRef(null);
  const [reactFlowInstance, setReactFlowInstance] = useState(null);

  useEffect(() => {
    console.log("useEffect nodes: ", nodes);
    console.log("useEffect edges: ", edges);
  }, [nodes]);

  const onConnect = useCallback((connection) =>
    setEdges((eds) =>
      addEdge(connection, eds)
    ));

  const onDragOver = useCallback((event) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event) => {
      event.preventDefault();

      const reactFlowBounds = reactFlowWrapper.current.getBoundingClientRect();
      const type = event.dataTransfer.getData('application/reactflow');

      // check if the dropped element is valid
      if (typeof type === 'undefined' || !type) {
        return;
      }

      const position = reactFlowInstance.project({
        x: event.clientX - 250,
        y: event.clientY - reactFlowBounds.top,
      });
      const newNode = {
        id: uuidv4(),
        type,
        position,
        data: { label: `${type} node` },
      };
      console.log("New node: " + newNode);

      setNodes((nds) => nds.concat(newNode));
      console.log("nodes: ", nodes)
    },
    [reactFlowInstance]
  );


  function OSC({ id, data }) {

    const onChange = useCallback((evt) => {
      console.log(evt.target.value);
    }, []);
  
    return (
      <div className={tw("rounded-md bg-white shadow-xl")}>
        <p
          className={tw("rounded-t-md px-2 py-1 bg-pink-500 text-white text-sm")}
        >
          Osc
        </p>
  
        <label className={tw("flex flex-col px-2 py-1")}>
          <p className={tw("text-xs font-bold mb-2")}>Frequency</p>
          <input
            className="nodrag"
            type="range"
            min="10"
            max="1000"
            value={data.frequency}
            onChange={onChange}
          />
          <p className={tw("text-right text-xs")}>{data.frequency} Hz</p>
        </label>
  
        <hr className={tw("border-gray-200 mx-2")} />
  
        <label className={tw("flex flex-col px-2 pt-1 pb-4")}>
          <p className={tw("text-xs font-bold mb-2")}>Waveform</p>
          <select className="nodrag" value={data.type} onChange={onChange}>
            <option value="sine">sine</option>
            <option value="triangle">triangle</option>
            <option value="sawtooth">sawtooth</option>
            <option value="square">square</option>
          </select>
        </label>
  
        <Handle className={tw("w-2 h-2")} type="source" position="bottom" onChange={onChange} />
      </div>
    );
  }
  

  return (
    <>
      <div className="dndflow">
        <ReactFlowProvider>
          <div style={{ width: '60vw', height: '80vh', float: 'left' }} className="reactflow-wrapper" ref={reactFlowWrapper}>
            <ReactFlow
          // nodeTypes={nodeTypes}
          // nodes={store.nodes}
          // edges={store.edges}
          // onNodesChange={store.onNodesChange}
          // onEdgesChange={store.onEdgesChange}
          // onConnect={store.addEdge}
              nodes={nodes}
              edges={edges}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onInit={setReactFlowInstance}
              onDrop={onDrop}
              onDragOver={onDragOver}
              nodeTypes={nodeTypes}
              fitView
            >
              <Controls />
              <MiniMap style={minimapStyle} zoomable pannable />
              <Background color="#aaa" gap={16} />
            </ReactFlow>
          </div>
          <div style={{ width: '20vw', float: 'right' }} className="reactflow-wrapper" ref={reactFlowWrapper}>
            <Sidebar />
          </div>
        </ReactFlowProvider>
      </div>
    </>
  )
}

export default Flowgraph




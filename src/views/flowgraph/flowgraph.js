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

const nodeTypes = { nodeTypeActionTalk: NodeActionTalk };

const Flowgraph = () => {
  console.log("Flowgraph");





  const initialNodes = [
    { id: uuidv4(), type: 'input', data: { label: 'Start' }, position: { x: 250, y: 0 } },
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


  return (
    <>
      <div className="dndflow">
        <ReactFlowProvider>
          <div style={{ width: '60vw', height: '80vh', float: 'left' }} className="reactflow-wrapper" ref={reactFlowWrapper}>
            <ReactFlow
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

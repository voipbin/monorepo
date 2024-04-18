import ReactFlow, { ReactFlowProvider } from 'reactflow';
import { useCallback } from 'react';
import { Handle, Position } from 'reactflow';

const handleStyle = { left: 10 };

export function NodeActionTalk({ data, isConnectable }) {
    const onChange = useCallback((evt) => {
    console.log(evt.target.value);
  }, []);

  // <div className="node-action-talk">
    return (
        <>
            {/* <ReactFlowProvider> */}

            {/* <Handle type="target"
                // position={Position.Left}
                onConnect={(params) => console.log('handle onConnect', params)}
                isConnectable={true}
            /> */}
            <div>
                <label htmlFor="text">Text:</label>
                <input id="text" name="text" onChange={onChange} className="nodrag" />
                <br></br>
                <label htmlFor="text">Gender:</label>
                <input id="text" name="text" onChange={onChange} className="nodrag" />
                <br></br>
                <label htmlFor="text">Language:</label>
                <input id="text" name="text" onChange={onChange} className="nodrag" />
                <br></br>
                <label htmlFor="text">Digits Handle:</label>
                <input id="text" name="text" onChange={onChange} className="nodrag" />
            </div>
            {/* <Handle
                type="source"
                // position={Position.Right}
                id="a"
                style={handleStyle}
                isConnectable={true}
                /> */}
                {/* </ReactFlowProvider> */}
        </>
    );
}


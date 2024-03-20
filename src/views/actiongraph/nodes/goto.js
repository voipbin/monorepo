import React, { useState, useRef } from 'react';
import { Handle, Position } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

import './styles.css';

// target_id: emptyID,
// loop_count: 1,
const selector = (id) => (store) => ({
  setTargetID: (e) => store.updateNode(id, { target_id: e.target.value }),
  setLoopCount: (e) => store.updateNode(id, { loop_count: e.target.value }),
});


// style presets
const handleStyle = {top: 60};

export default function Goto({ id, data }) {
  const { edges, setLoopCount, setTargetID} = useStore(selector(id), shallow);
  
  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";
  const handleSourceTargetID = id + "-source_target";

  const updateTargetID = (e) => {
    console.log("updateTargetIDs. event: %o", e);

    var targetID = "00000000-0000-0000-0000-000000000000"; 
    for (let i = 0; i < edges.length; i++) {
      const edge = edges[i];
      if (edge.sourceHandle == handleSourceTargetID) {
        targetID = edge.target;
        break;
      }
    }

    setTargetID(targetID);
  }

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Goto</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Target</p>
        <Handle id={handleSourceTargetID} className={tw('w-2 h-2')} type="source" position="right" onChange={updateTargetID} style={handleStyle}/>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Loop Count</p>
        <input type="number" className="nodrag" value={data.loop_count} onChange={setLoopCount} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />

    </div>
  );
}

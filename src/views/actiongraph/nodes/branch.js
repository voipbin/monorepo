import React, { useState, useRef } from 'react';
import { Handle, Position } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

import './styles.css';

// "variable": "<string>",
// "default_target_id": "<string>",
// "target_ids": {
//     "<string>": <string>,
// }
const selector = (id) => (store) => ({
  edges: store.edges,

  setVariable: (e) => store.updateNode(id, { variable: e.target.value }),
  setDefaultTargetID: (e) => store.updateNode(id, { default_target_id: e.target.value }),
  setTargetIDs: (targetIDs) => store.updateNode(id, { target_ids: targetIDs}),
});


// style presets
const handleStyles = [{top: 190}];
for (let i = 0; i < 100; i++) {
  const tmp = {top: handleStyles[i].top + 77}
  handleStyles.push(tmp);    
}

export default function Branch({ id, data }) {
  const { edges, setVariable, setDefaultTargetID, setTargetIDs} = useStore(selector(id), shallow);
  const [targetTimes, setTargetTimes] = useState(Object.keys(data.target_ids).length);
  
  const handleTargetID = id + "-target";
  const handleSourceDefaultID = id + "-source";

  console.log("Branch: %o", data);
  console.log("keys: %o, length: %o", Object.keys(data.target_ids), Object.keys(data.target_ids).length);

  const onClickAdd = (e) => {
    console.log("onClickAdd. targetIDs: %o, event: %o", data.target_ids, e);

    setTargetIDs({
      ...data.target_ids,
      "target": "00000000-0000-0000-0000-000000000000",
    });
    setTargetTimes(targetTimes + 1);
  };

  const updateTargetIDs = (e) => {
    console.log("updateTargetIDs. event: %o", e);

    var targetIDs = {};
    for (let i = 0; i < targetTimes; i++) {
      const tmpID = handleSourceDefaultID + "_" + i;

      console.log("Finding element. tmpID: %s", tmpID);
      var tmp = document.getElementById(tmpID);      
      if (tmp === undefined || tmp === null) {
        continue;
      }
      console.log("Found tmp info. tmp: %o", tmp);

      var targetID = "00000000-0000-0000-0000-000000000000"; 
      for (let i = 0; i < edges.length; i++) {
        const edge = edges[i];
        if (edge.sourceHandle == tmpID) {
          targetID = edge.target;
          break;
        }
      }

      targetIDs[tmp.value] = targetID;
    }

    console.log("Printing result. TargetIDs: %o", targetIDs);
    setTargetIDs(targetIDs);
  }

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Branch</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Variable</p>
        <input className="nodrag" value={data.variable} onChange={setVariable} />
      </label>

      <div className="nodrag">
        <button type="button" className="button-9" onClick={onClickAdd}>Add targets</button>
      </div>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Default target</p>
        <Handle id={handleSourceDefaultID} className={tw('w-2 h-2')} type="source" position="right" onChange={setDefaultTargetID} style={handleStyles[0]}/>
      </label>

      {Array.apply(null, { length: Object.keys(data.target_ids).length }).map((e, i) => (
        <>
          <hr className={tw('rounded-md bg-white shadow-xl border-black-100 mx-2')} />
          <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
            <p className={tw('text-xs font-bold mb-2')}>Target</p>
            <Handle id={handleSourceDefaultID + "_" + i} className={tw('w-2 h-2')} type="source" position="right" style={handleStyles[i+1]}/>
            <input id={handleSourceDefaultID + "_" + i} className="nodrag" value={Object.keys(data.target_ids)[i]} onChange={updateTargetIDs}/>
          </label>
        </>
      ))}
    </div>
  );
}

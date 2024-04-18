import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

// reason: "normal",
// reference_id: emptyID,
const selector = (id) => (store) => ({
  setReason: (e) => store.updateNode(id, { reason: e.target.value }),
  setReferenceID: (e) => store.updateNode(id, { reference_id: e.target.value }),
});

export default function Hangup({ id, data }) {
  const { setReason, setReferenceID } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Hangup</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Reason</p>
        <select className="nodrag" value={data.reason} onChange={setReason}>
          <option value="normal">Normal</option>
          <option value="failed">Failed</option>
          <option value="busy">Busy</option>
          <option value="cancel">Cancel</option>
          <option value="timeout">Timeout</option>
          <option value="noanswer">NoAnswer</option>
          <option value="dialout">Dialout</option>
          <option value="amd">AMD</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Reference ID</p>
        <input className="nodrag" value={data.reference_id} onChange={setReferenceID} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';


// confbridge_id: emptyID,
const selector = (id) => (store) => ({
  setConfbridgeID: (e) => store.updateNode(id, { confbridge_id: e.target.value }),
});

export default function ConfbridgeJoin({ id, data }) {
  const { setConfbridgeID } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Confbridge Join</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Confbridge ID</p>
        <input className="nodrag" value={data.confbridge_id} onChange={setConfbridgeID} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

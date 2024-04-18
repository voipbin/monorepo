import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';


// duration: 10000,
const selector = (id) => (store) => ({
  setDuration: (e) => {
    try {
      store.updateNode(id, { duration: e.target.value });
    } catch (e) {
      // do nothing
    }
  },
});

export default function Echo({ id, data }) {
  const { setDuration } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Echo</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Duration</p>
        <input type="number" className="nodrag" value={data.duration} onChange={setDuration} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

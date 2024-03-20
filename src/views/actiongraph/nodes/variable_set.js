import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

// key: "variable key",
// value: "variable value",
const selector = (id) => (store) => ({
  setKey: (e) => store.updateNode(id, { key: e.target.value }),
  setValue: (e) => store.updateNode(id, { value: e.target.value }),
});

export default function TranscribeStart({ id, data }) {
  const { setKey, setValue } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Variable Set</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Key</p>
        <input className="nodrag" value={data.key} onChange={setKey} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Value</p>
        <input className="nodrag" value={data.value} onChange={setValue} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

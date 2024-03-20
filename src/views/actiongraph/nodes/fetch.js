import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

// event_url: "example.com",
// event_method: "POST",
const selector = (id) => (store) => ({
  setEventURL: (e) => store.updateNode(id, { event_url: e.target.value }),
  setEventMethod: (e) => store.updateNode(id, { event_method: e.target.value }),
});

export default function Fetch({ id, data }) {
  const { setEventURL, setEventMethod } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Fetch</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Event URL</p>
        <input className="nodrag" value={data.event_url} onChange={setEventURL} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Event Method</p>
        <select className="nodrag" value={data.event_method} onChange={setEventMethod}>
          <option value="POST">POST</option>
          <option value="GET">GET</option>
          <option value="PUT">PUT</option>
          <option value="DELETE">DELETE</option>
        </select>
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

// sync: false,
// uri: "example.com/webhook/send/uri",
// method: "POST",
// data_type: "application/json",
// data: "Hello, world",
const selector = (id) => (store) => ({
  setSync: (e) => store.updateNode(id, { sync: e.target.value }),
  setURI: (e) => store.updateNode(id, { uri: e.target.value }),
  setMethod: (e) => store.updateNode(id, { method: e.target.value }),
  setDataType: (e) => store.updateNode(id, { data_type: e.target.value }),
  setData: (e) => store.updateNode(id, { data: e.target.value }),
});

export default function WebhookSend({ id, data }) {
  const { setSync, setURI, setMethod, setDataType, setData } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Webhook Send</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Sync</p>
        <label class="switch">
          <input type="checkbox" className="nodrag" value={data.sync} onChange={setSync} id="switch"/>
          <span class="slider round"></span>
        </label>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>URI</p>
        <input className="nodrag" value={data.uri} onChange={setURI} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Method</p>
        <select className="nodrag" value={data.method} onChange={setMethod}>
          <option value="POST">POST</option>
          <option value="GET">GET</option>
          <option value="PUT">PUT</option>
          <option value="DELETE">DELETE</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Data Type</p>
        <input className="nodrag" value={data.data_type} onChange={setDataType} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Data</p>
        <input className="nodrag" value={data.data} onChange={setData} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

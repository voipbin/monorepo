import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';


// source: {
//   "type": "tel",
//   "target": ""
// },
// destinations: [
//   {
//     "type": "tel",
//     "target": ""
//   }
// ],
// early_media: false,
// relay_reason: false,
const selector = (id) => (store) => ({
  setSource: (e) => {
    try {
      store.updateNode(id, { source: JSON.parse(e.target.value) });
    } catch (e) {
      // do nothing
    }
  },
  setDestinations: (e) => {
    try {
      store.updateNode(id, { source: JSON.parse(e.target.value) });
    } catch (e) {
      // do nothing
    }
  },
  setEarlyMedia: (e) => {
    store.updateNode(id, { early_media: e.target.value })
  },
  setRelayReason: (e) => {
    store.updateNode(id, { relay_reason: e.target.value })
  },
});

export default function Connect({ id, data }) {
  const { setSource, setDestinations, setEarlyMedia, setRelayReason } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Connect</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Source</p>
        <textarea raws={5} className="nodrag" value={JSON.stringify(data.source, null, 2)} onChange={setSource} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Destinations</p>
        <textarea raws={10} className="nodrag" value={JSON.stringify(data.destinations, null, 2)} onChange={setDestinations} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Early Media</p>
        <label class="switch">
          <input type="checkbox" className="nodrag" value={data.early_media} onChange={setEarlyMedia} id="switch"/>
          <span class="slider round"></span>
        </label>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Relay Reason</p>
        <label class="switch">
          <input type="checkbox" className="nodrag" value={data.relay_reason} onChange={setRelayReason} id="switch"/>
          <span class="slider round"></span>
        </label>
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

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
// text: "hello world",
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
  setText: (e) => store.updateNode(id, { text: e.target.value }),
});

export default function MessageSend({ id, data }) {
  const { setSource, setDestinations, setText } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Message Send</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Source</p>
        <textarea raws={5} className="nodrag" value={JSON.stringify(data.source, null, 2)} onChange={setSource} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Destinations</p>
        <textarea raws={10} className="nodrag" value={JSON.stringify(data.destinations, null, 2)} onChange={setDestinations} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Text</p>
        <input className="nodrag" value={data.text} onChange={setText} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';


// stream_urls: [
//   "example.com/play/file/url",
// ],
const selector = (id) => (store) => ({
  setStreamURLs: (e) => {
    try {
      store.updateNode(id, { stream_urls: JSON.parse(e.target.value) });
    } catch (e) {
      // do nothing
    }
  },
});

export default function Play({ id, data }) {
  const { setStreamURLs } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Play</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Steam URLs</p>
        <textarea raws={5} className="nodrag" value={JSON.stringify(data.stream_urls, null, 2)} onChange={setStreamURLs} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';


// digits: "123",
// duration: 1000,
// interval: 500,
const selector = (id) => (store) => ({
  setDigits: (e) => {
    try {
      store.updateNode(id, { digits: e.target.value });
    } catch (e) {
      // do nothing
    }
  },
  setDuration: (e) => {
    try {
      store.updateNode(id, { duration: e.target.value });
    } catch (e) {
      // do nothing
    }
  },
  setInterval: (e) => {
    try {
      store.updateNode(id, { interval: e.target.value });
    } catch (e) {
      // do nothing
    }
  },
});

export default function DigitsSend({ id, data }) {
  const { setDigits, setDuration, setInterval } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Digits Send</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Digits</p>
        <input className="nodrag" value={data.digits} onChange={setDigits} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Duration</p>
        <input type="number" className="nodrag" value={data.duration} onChange={setDuration} />
      </label>


      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Interval</p>
        <input type="number" className="nodrag" value={data.interval} onChange={setInterval} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

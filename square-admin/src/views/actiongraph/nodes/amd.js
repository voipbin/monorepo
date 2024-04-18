import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

import './styles.css';

const selector = (id) => (store) => ({
  setMachineHandle: (e) => store.updateNode(id, { machine_handle: e.target.value }),
  setAsync: (e) => store.updateNode(id, { async: e.target.value }),
});

export default function AMD({ id, data }) {
  const { setMachineHandle, setAsync} = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>AMD</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Machine handle</p>
        <select className="nodrag" value={data.machine_handle} onChange={setMachineHandle}>
          <option value="hangup">Hangup</option>
          <option value="continue">Continue</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Async</p>

        <label class="switch">
          <input type="checkbox" className="nodrag" value={data.async} onChange={setAsync} id="switch"/>
          <span class="slider round"></span>
        </label>
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

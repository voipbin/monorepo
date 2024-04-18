import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

const selector = (id) => (store) => ({
  setGender: (e) => store.updateNode(id, { gender: e.target.value }),
  setLanguage: (e) => store.updateNode(id, { language: e.target.value }),
  setDigitsHandle: (e) => store.updateNode(id, { digits_handle: e.target.value }),
  setText: (e) => store.updateNode(id, { text: e.target.value }),
});

export default function Talk({ id, data }) {
  const { setGender, setLanguage, setDigitsHandle, setText } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Talk</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Gender</p>
        <select className="nodrag" value={data.gender} onChange={setGender}>
          <option value="male">male</option>
          <option value="female">female</option>
          <option value="neutral">neutral</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Lanugage</p>
        <input className="nodrag" value={data.language} onChange={setLanguage} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Digits handle</p>
        <select className="nodrag" value={data.digits_handle} onChange={setDigitsHandle}>
          <option value="">none</option>
          <option value="next">next</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Text</p>
        <input className="nodrag" value={data.text} onChange={setText} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

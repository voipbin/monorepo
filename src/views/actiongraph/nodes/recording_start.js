import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

// format: "wav",
// end_of_silence: 0,
// end_of_key: "",
// duration: 0,
// beep_start: false,
const selector = (id) => (store) => ({
  setFormat: (e) => store.updateNode(id, { format: e.target.value }),
  setEndOfSilence: (e) => store.updateNode(id, { end_of_silence: e.target.value }),
  setEndOfKey: (e) => store.updateNode(id, { end_of_key: e.target.value }),
  setDuration: (e) => store.updateNode(id, { duration: e.target.value }),
  setBeepStart: (e) => store.updateNode(id, { beep_start: e.target.value }),
});

export default function RecordingStart({ id, data }) {
  const { setFormat, setEndOfSilence, setEndOfKey, setDuration, setBeepStart } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Recording Start</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Format</p>
        <select className="nodrag" value={data.format} onChange={setFormat}>
          <option value="wav">WAV</option>
          <option value="mp3">MP3</option>
          <option value="ogg">OGG</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>End Of Silence</p>
        <input type="number" className="nodrag" value={data.end_of_silence} onChange={setEndOfSilence} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>End Of Key</p>
        <input className="nodrag" value={data.end_of_key} onChange={setEndOfKey} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Duration</p>
        <input type="number" className="nodrag" value={data.duration} onChange={setDuration} />
      </label>


      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Beep Start</p>

        <label class="switch">
          <input type="checkbox" className="nodrag" value={data.beep_start} onChange={setBeepStart} id="switch"/>
          <span class="slider round"></span>
        </label>
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

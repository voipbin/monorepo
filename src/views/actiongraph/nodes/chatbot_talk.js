import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';


// chatbot_id: emptyID,
// gender: "male",
// language: "en-US",
// duration: 3600
const selector = (id) => (store) => ({
  setChatbotID: (e) => store.updateNode(id, { chatbot_id: e.target.value }),
  setGender: (e) => store.updateNode(id, { gender: e.target.value }),
  setLanguage: (e) => store.updateNode(id, { language: e.target.value }),
  setDuration: (e) => store.updateNode(id, { duration: e.target.value }),
});

export default function ChatbotTalk({ id, data }) {
  const { setChatbotID, setGender, setLanguage, setDuration } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Chatbot Talk(AI)</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Chatbot ID</p>
        <input className="nodrag" value={data.chatbot_id} onChange={setChatbotID} />
      </label>


      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Gender</p>
        <select className="nodrag" value={data.gender} onChange={setGender}>
          <option value="male">male</option>
          <option value="female">female</option>
          <option value="neutral">neutral</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Language</p>
        <input className="nodrag" value={data.language} onChange={setLanguage} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Duration</p>
        <input type="number" className="nodrag" value={data.duration} onChange={setDuration} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

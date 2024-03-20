import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';


// conversation_id: emptyID,
// text: "hello world",
// sync: false,
const selector = (id) => (store) => ({
  setConversationID: (e) => {
    try {
      store.updateNode(id, { conversation_id: e.target.value });
    } catch (e) {
      // do nothing
    }
  },
  setText: (e) => {
    try {
      store.updateNode(id, { text: e.target.value });
    } catch (e) {
      // do nothing
    }
  },
  setSync: (e) => {
    try {
      store.updateNode(id, { sync: e.target.value });
    } catch (e) {
      // do nothing
    }
  },
});

export default function ConversationSend({ id, data }) {
  const { setConversationID, setText, setSync } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Conversation Send</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Conversation ID</p>
        <input className="nodrag" value={data.conversation_id} onChange={setConversationID} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Text</p>
        <input className="nodrag" value={data.text} onChange={setText} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Sync</p>

        <label class="switch">
          <input type="checkbox" className="nodrag" value={data.sync} onChange={setSync} id="switch"/>
          <span class="slider round"></span>
        </label>
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

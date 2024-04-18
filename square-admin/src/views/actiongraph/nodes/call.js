import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';


// source: {},
// destinations: [],
// flow_id: emptyID,
// actions: [],
// chained: false,
// early_execution: false,
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
  setFlowID: (e) => store.updateNode(id, { flow_id: e.target.value }),
  setActions: (e) => {
    try {
      store.updateNode(id, { source: JSON.parse(e.target.value) });
    } catch (e) {
      // do nothing
    }
  },
  setChained: (e) => {
    store.updateNode(id, { chained: e.target.value })
  },
  setEarlyExecution: (e) => {
    store.updateNode(id, { early_execution: e.target.value })
  },
});

export default function Call({ id, data }) {
  const { setSource, setDestinations, setFlowID, setActions, setChained, setEarlyExecution } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>Call</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Source</p>
        <textarea raws={5} className="nodrag" value={JSON.stringify(data.source, null, 2)} onChange={setSource} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Destinations</p>
        <textarea raws={10} className="nodrag" value={JSON.stringify(data.destinations, null, 2)} onChange={setDestinations} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Flow ID</p>
        <input className="nodrag" value={data.flow_id} onChange={setFlowID} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Actions</p>
        <textarea raws={10} className="nodrag" value={JSON.stringify(data.actions, null, 2)} onChange={setActions} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Chained</p>

        <label class="switch">
          <input type="checkbox" className="nodrag" value={data.chained} onChange={setChained} id="switch"/>
          <span class="slider round"></span>
        </label>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Early Execution</p>

        <label class="switch">
          <input type="checkbox" className="nodrag" value={data.early_execution} onChange={setEarlyExecution} id="switch"/>
          <span class="slider round"></span>
        </label>
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

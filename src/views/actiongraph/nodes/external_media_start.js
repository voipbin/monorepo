import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

// external_host: 'example.com',
// encapsulation: 'rtp',
// transport: 'udp',
// connection_type: 'client',
// format: 'ulaw',
// direction: 'both',
// data: '',
const selector = (id) => (store) => ({
  setExternalHost: (e) => store.updateNode(id, { external_host: e.target.value }),
  setEncapsulation: (e) => store.updateNode(id, { encapsulation: e.target.value }),
  setTransport: (e) => store.updateNode(id, { transport: e.target.value }),
  setConnectionType: (e) => store.updateNode(id, { connection_type: e.target.value }),
  setFormat: (e) => store.updateNode(id, { format: e.target.value }),
  setDirection: (e) => store.updateNode(id, { direction: e.target.value }),
  setData: (e) => store.updateNode(id, { data: e.target.value }),
});

export default function ExternalMediaStart({ id, data }) {
  const { setExternalHost, setEncapsulation, setTransport, setConnectionType, setFormat, setDirection, setData } = useStore(selector(id), shallow);

  const handleTargetID = id + "-target";
  const handleSourceID = id + "-source";

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <Handle id={handleTargetID} className={tw('w-2 h-2')} type="target" position="top" />

      <p className={tw('rounded-t-md px-2 py-1 bg-blue-500 text-white text-sm')}>External Media Start</p>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>External Host</p>
        <input className="nodrag" value={data.external_host} onChange={setExternalHost} />
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Encapsulation</p>
        <select className="nodrag" value={data.encapsulation} onChange={setEncapsulation}>
          <option value="rtp">RTP</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Transport</p>
        <select className="nodrag" value={data.transport} onChange={setTransport}>
          <option value="udp">UDP</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Connection Type</p>
        <select className="nodrag" value={data.connection_type} onChange={setConnectionType}>
          <option value="client">Client</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Format</p>
        <select className="nodrag" value={data.format} onChange={setFormat}>
          <option value="ulaw">Ulaw</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Direction</p>
        <select className="nodrag" value={data.direction} onChange={setDirection}>
          <option value="both">Both</option>
        </select>
      </label>

      <label className={tw('flex flex-col px-2 pt-1 pb-4')}>
        <p className={tw('text-xs font-bold mb-2')}>Data</p>
        <input className="nodrag" value={data.data} onChange={setData} />
      </label>

      <Handle id={handleSourceID} className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

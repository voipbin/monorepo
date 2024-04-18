import React from 'react';
import { Handle } from 'reactflow';
import { shallow } from 'zustand/shallow';
import { tw } from 'twind';
import { useStore } from '../store';

export default function Start({ id, data }) {

  return (
    <div className={tw('rounded-md bg-white shadow-xl')}>
      <p className={tw('rounded-t-md px-2 py-1 bg-pink-500 text-white text-sm')}>Start</p>
      <br />

      <Handle className={tw('w-2 h-2')} type="source" position="bottom" />
    </div>
  );
}

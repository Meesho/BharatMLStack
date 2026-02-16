import React from 'react';

export default function Root({ children }) {
  return (
    <>
      <div className="gradient-bg-global">
        <div className="gradient-orb-global orb-global-1" />
        <div className="gradient-orb-global orb-global-2" />
        <div className="gradient-orb-global orb-global-3" />
      </div>
      {children}
    </>
  );
}

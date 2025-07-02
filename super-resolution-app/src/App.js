import React, { useState } from 'react';
import Dashboard from './components/Dashboard';
import SuperResolutionImagesTable from './components/SuperResolutionImagesTable';

function App() {
  const [view, setView] = useState('dashboard');

  return (
    <div>
      <nav>
      <button
        onClick={() => setView('dashboard')}
        style={{ fontWeight: view === 'dashboard' ? 'bold' : 'normal' }}
      >
        📈 Dashboard
      </button>
      <button
        onClick={() => setView('superResolutionImages')}
        style={{ fontWeight: view === 'superResolutionImages' ? 'bold' : 'normal' }}
      >
        🗂️ SuperResolutionImages
      </button>
      </nav>
      <hr />
      {view === 'dashboard' ? <Dashboard /> : <SuperResolutionImagesTable />}
    </div>
  );
}

export default App;

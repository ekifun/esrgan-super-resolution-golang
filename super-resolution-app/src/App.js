import React, { useState } from 'react';
import Dashboard from './components/Dashboard';
import HistoryTable from './components/HistoryTable';

function App() {
  const [view, setView] = useState('dashboard');

  return (
    <div>
      <nav>
        <button onClick={() => setView('dashboard')}>📈 Dashboard</button>
        <button onClick={() => setView('history')}>🗂️ History</button>
      </nav>
      <hr />
      {view === 'dashboard' ? <Dashboard /> : <HistoryTable />}
    </div>
  );
}

export default App;

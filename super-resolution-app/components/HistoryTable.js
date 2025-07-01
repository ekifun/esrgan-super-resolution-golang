import React, { useEffect, useState } from 'react';

const HistoryTable = () => {
  const [history, setHistory] = useState([]);

  useEffect(() => {
    fetch('/get-status')
      .then(res => res.json())
      .then(data => {
        const processed = data.processed || [];
        setHistory(processed);
      })
      .catch(err => console.error('Error fetching history:', err));
  }, []);

  return (
    <div>
      <h2>🗂️ Super-Resolution History</h2>
      <table border="1" cellPadding="8">
        <thead>
          <tr>
            <th>Original Image URL</th>
            <th>Upscaled Image URL</th>
            <th>Topic Name</th>
          </tr>
        </thead>
        <tbody>
          {history.map((topic, idx) => (
            <tr key={idx}>
              <td><a href={topic.imageURL} target="_blank" rel="noreferrer">View Original</a></td>
              <td><a href={topic.upscaledURL} target="_blank" rel="noreferrer">Download Upscaled</a></td>
              <td>{topic.name}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default HistoryTable;

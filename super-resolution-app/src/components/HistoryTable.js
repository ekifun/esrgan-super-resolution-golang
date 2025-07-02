import React, { useEffect, useState } from 'react';

const HistoryTable = () => {
  const [history, setHistory] = useState([]);

  useEffect(() => {
    fetch('/get-status')
      .then(res => res.json())
      .then(data => {
        const processed = (data.processed || []).map((item) => {
          try {
            // Handle case where the entire item is a JSON string
            if (typeof item === 'string' && item.startsWith('{')) {
              return JSON.parse(item);
            }
        
            // Handle case where `item.name` is a stringified JSON
            if (item.name && typeof item.name === 'string' && item.name.startsWith('{')) {
              return JSON.parse(item.name);
            }
        
            // Assume already structured object
            return item;
          } catch (e) {
            console.error("❌ Failed to parse topic:", e);
            return {};
          }
        });        
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
          {history.map((topic, idx) => {
            console.log("🧪 topic:", topic);
            return (
              <tr key={idx}>
                <td>
                  {topic.imageURL ? (
                    <a
                      href={topic.imageURL}
                      target="_blank"
                      rel="noreferrer"
                      download
                    >
                      Download Original
                    </a>
                  ) : (
                    <span>N/A</span>
                  )}
                </td>
                <td>
                  {topic.upscaledURL ? (
                    <a
                      href={topic.upscaledURL}
                      target="_blank"
                      rel="noreferrer"
                      download
                    >
                      Download Upscaled
                    </a>
                  ) : (
                    <span>Pending</span>
                  )}
                </td>
                <td>{topic.name || 'Unknown'}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};

export default HistoryTable;

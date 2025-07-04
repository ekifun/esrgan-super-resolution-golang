import React, { useEffect, useState, useRef } from 'react';

function isValidUrl(str) {
  try {
    new URL(str);
    return true;
  } catch (_) {
    return false;
  }
}

function Dashboard() {
  const [processedTopics, setProcessedTopics] = useState([]);
  const [processingTopics, setProcessingTopics] = useState([]);
  const [topicName, setTopicName] = useState('');
  const [imageURL, setImageURL] = useState('');
  const topicMapRef = useRef(new Map());

  useEffect(() => {
    fetch('/get-status')
      .then((res) => res.json())
      .then((data) => {
        setProcessedTopics(data.processed || []);
        const processing = (data.processing || []).map(t => ({
          ...t,
          progress: parseInt(t.progress, 10)
        }));
        setProcessingTopics(processing);
        const map = new Map();
        processing.forEach((t) => map.set(t.name, t));
        topicMapRef.current = map;
      })
      .catch((err) => console.error('Initial fetch error:', err));

    const eventSource = new EventSource("http://13.57.143.121:5001/events");
    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'progress') {
          const progressVal = parseInt(data.progress, 10);
          setProcessingTopics((prev) => {
            const index = prev.findIndex((t) => t.name === data.topic_id);
            if (index !== -1) {
              const updated = [...prev];
              updated[index] = { ...updated[index], progress: progressVal };
              return updated;
            } else {
              return [...prev, { name: data.topic_id, progress: progressVal }];
            }
          });
        } else if (data.type === 'complete') {
          setProcessingTopics((prev) => prev.filter((t) => t.name !== data.topic_id));
          setProcessedTopics((prev) => [
            ...prev,
            {
              name: data.topic_id,
              imageURL: data.imageURL,
              upscaledURL: data.upscaledURL || data.result,
            },
          ]);
        }
      } catch (e) {
        console.error('❌ SSE parse error:', e);
      }
    };

    eventSource.onerror = (err) => {
      console.error('SSE error:', err);
      eventSource.close();
    };

    return () => eventSource.close();
  }, []);

  const handleSubmit = (e) => {
    e.preventDefault();
    const trimmedName = topicName.trim();
    const newTopic = { name: trimmedName, progress: 0 };
    setProcessingTopics((prev) => [...prev, newTopic]);
    topicMapRef.current.set(trimmedName, newTopic);

    fetch('/submit-topic', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ topicName: trimmedName, imageURL }),
    })
      .then((res) => res.json())
      .then(() => {
        setTopicName('');
        setImageURL('');
      })
      .catch((err) => {
        console.error('Submission error:', err);
        setProcessingTopics((prev) => prev.filter((t) => t.name !== trimmedName));
        topicMapRef.current.delete(trimmedName);
      });
  };

  return (
    <div style={{ padding: '40px', fontFamily: 'Arial, sans-serif', maxWidth: '900px', margin: 'auto' }}>
      <h1>🎉 Processed Topics</h1>
      <table style={{ width: '100%', borderCollapse: 'collapse', marginBottom: '40px' }}>
        <thead>
          <tr>
            <th style={thStyle}>Topic</th>
            <th style={thStyle}>Original</th>
            <th style={thStyle}>Upscaled</th>
          </tr>
        </thead>
        <tbody>
          {processedTopics.map((topic, i) => (
            <tr key={i} style={{ borderBottom: '1px solid #ccc' }}>
              <td style={tdStyle}>{topic.name || 'N/A'}</td>
              <td style={tdStyle}>
                {isValidUrl(topic.imageURL) ? (
                  <a href={topic.imageURL} target="_blank" rel="noreferrer" download>
                    <img src={topic.imageURL} alt="original" style={imgThumb} />
                  </a>
                ) : (
                  <span style={{ color: 'gray' }}>N/A</span>
                )}
              </td>
              <td style={tdStyle}>
                {isValidUrl(topic.upscaledURL) ? (
                  <a href={topic.upscaledURL} target="_blank" rel="noreferrer" download>
                    <img src={topic.upscaledURL} alt="upscaled" style={imgThumb} />
                  </a>
                ) : (
                  <span style={{ color: 'gray' }}>N/A</span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <h1>🔄 Processing Topics</h1>
      <div style={{ marginBottom: '40px' }}>
        {processingTopics.map((topic, i) => (
          <div key={i} style={{ marginBottom: '20px' }}>
            <strong>{topic.name}</strong>
            <div style={progressWrapper}>
              <div
                style={{
                  ...progressBar,
                  width: `${topic.progress}%`,
                  backgroundColor: topic.progress === 100 ? '#4caf50' : '#2196f3',
                }}
              >
                {topic.progress}%
              </div>
            </div>
          </div>
        ))}
      </div>

      <h1>📤 Submit New Task</h1>
      <form onSubmit={handleSubmit} style={formStyle}>
        <input
          type="text"
          value={topicName}
          placeholder="Enter topic name"
          onChange={(e) => setTopicName(e.target.value)}
          style={inputStyle}
          required
        />
        <input
          type="url"
          value={imageURL}
          placeholder="Enter image URL"
          onChange={(e) => setImageURL(e.target.value)}
          style={inputStyle}
          required
        />
        <button type="submit" style={submitStyle}>Submit</button>
      </form>
    </div>
  );
}

// Styles
const thStyle = {
  textAlign: 'left',
  padding: '10px',
  backgroundColor: '#f4f4f4',
};

const tdStyle = {
  padding: '10px',
  verticalAlign: 'middle',
};

const imgThumb = {
  width: '120px',
  borderRadius: '6px',
  border: '1px solid #ccc',
};

const progressWrapper = {
  width: '100%',
  backgroundColor: '#eee',
  borderRadius: '5px',
  overflow: 'hidden',
  height: '24px',
  marginTop: '8px',
};

const progressBar = {
  height: '100%',
  color: 'white',
  textAlign: 'center',
  lineHeight: '24px',
  fontSize: '12px',
  transition: 'width 0.4s ease',
};

const formStyle = {
  display: 'flex',
  flexDirection: 'column',
  gap: '12px',
  maxWidth: '500px',
};

const inputStyle = {
  padding: '10px',
  fontSize: '16px',
  borderRadius: '4px',
  border: '1px solid #ccc',
};

const submitStyle = {
  padding: '10px 16px',
  fontSize: '16px',
  backgroundColor: '#2196f3',
  color: 'white',
  border: 'none',
  borderRadius: '4px',
  cursor: 'pointer',
};

export default Dashboard;

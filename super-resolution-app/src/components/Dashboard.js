import React, { useEffect, useState, useRef } from 'react';

function Dashboard() {
  const [processedTopics, setProcessedTopics] = useState([]);
  const [processingTopics, setProcessingTopics] = useState([]);
  const [topicName, setTopicName] = useState('');
  const [imageURL, setImageURL] = useState('');

  // For fast lookup by topic name
  const topicMapRef = useRef(new Map());

  useEffect(() => {
    fetch('/get-status')
      .then((res) => res.json())
      .then((data) => {
        setProcessedTopics(data.processed || []);
        setProcessingTopics(data.processing || []);
        const map = new Map();
        (data.processing || []).forEach(t => map.set(t.name, t));
        topicMapRef.current = map;
      })
      .catch((err) => console.error('Initial fetch error:', err));

    const eventSource = new EventSource("http://13.57.143.121:5001/events");

    eventSource.onmessage = (event) => {
      console.log("📩 SSE Event Received:", event.data);  // <-- Add this log
    
      try {
        const data = JSON.parse(event.data);
    
        if (data.type === 'progress') {
          console.log("🟡 Progress Update:", data);  // Log progress-specific data
    
          setProcessingTopics((prev) => {
            const index = prev.findIndex((t) => t.name === data.topic_id);
            if (index !== -1) {
              const updated = [...prev];
              updated[index] = { ...updated[index], progress: data.progress };
              return updated;
            } else {
              return [...prev, { name: data.topic_id, progress: data.progress }];
            }
          });
        } else if (data.type === 'complete') {
          console.log("✅ Completion Update:", data);  // Log completion-specific data
    
          setProcessingTopics((prev) => prev.filter((t) => t.name !== data.topic_id));
          setProcessedTopics((prev) => [
            ...prev,
            {
              name: data.topic_id,
              imageURL: data.imageURL,
              upscaledURL: data.upscaledURL,
            },
          ]);
        }
      } catch (e) {
        console.error('❌ SSE message parse error:', e);
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
    <div style={{ padding: '20px', fontFamily: 'sans-serif' }}>
      <h1>Processed Topics</h1>
      <table>
        <thead>
          <tr>
            <th>Topic Name</th>
            <th>Original Image</th>
            <th>Upscaled Image</th>
          </tr>
        </thead>
        <tbody>
          {processedTopics.map((topic, i) => (
            <tr key={i}>
              <td>{topic.name}</td>
              <td><a href={topic.imageURL} target="_blank" rel="noreferrer">View</a></td>
              <td><a href={topic.upscaledURL} target="_blank" rel="noreferrer">View</a></td>
            </tr>
          ))}
        </tbody>
      </table>

      <h1>Processing Topics</h1>
      <div>
        {processingTopics.map((topic, i) => (
          <div key={i} style={{ marginBottom: '16px' }}>
            <strong>{topic.name}</strong>
            <div style={{
              backgroundColor: '#e0e0df',
              borderRadius: '4px',
              height: '20px',
              width: '100%',
              marginTop: '4px'
            }}>
              <div style={{
                width: `${topic.progress}%`,
                backgroundColor: topic.progress === 100 ? '#4caf50' : '#2196f3',
                height: '100%',
                borderRadius: '4px',
                textAlign: 'center',
                color: 'white',
                lineHeight: '20px',
                fontSize: '12px',
                transition: 'width 0.3s ease'
              }}>
                {topic.progress}%
              </div>
            </div>
          </div>
        ))}
      </div>


      <h1>Submit New Frame Upscaling Task</h1>
      <form onSubmit={handleSubmit}>
        <input
          type="text"
          value={topicName}
          placeholder="Enter video file name (topicName)"
          onChange={(e) => setTopicName(e.target.value)}
          required
        />
        <input
          type="url"
          value={imageURL}
          placeholder="Enter image URL to upscale"
          onChange={(e) => setImageURL(e.target.value)}
          required
        />
        <button type="submit">Submit</button>
      </form>
    </div>
  );
}

export default Dashboard;

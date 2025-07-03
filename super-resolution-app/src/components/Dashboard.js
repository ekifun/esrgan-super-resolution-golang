import React, { useEffect, useState } from 'react';

function Dashboard() {
  const [processedTopics, setProcessedTopics] = useState([]);
  const [processingTopics, setProcessingTopics] = useState([]);
  const [topicName, setTopicName] = useState('');
  const [imageURL, setImageURL] = useState('');

  useEffect(() => {
    // Initial fetch
    fetch('/get-status')
      .then((res) => res.json())
      .then((data) => {
        setProcessedTopics(data.processed || []);
        setProcessingTopics(data.processing || []);
      })
      .catch((err) => console.error('Initial fetch error:', err));

    // Setup SSE
    const eventSource = new EventSource("http://13.57.143.121:5001/events");

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);

        if (data.type === 'progress') {
          setProcessingTopics((prev) => {
            const exists = prev.find((t) => t.name === data.topic_id);
            if (exists) {
              return prev.map((t) =>
                t.name === data.topic_id ? { ...t, progress: data.progress } : t
              );
            } else {
              return [...prev, { name: data.topic_id, progress: data.progress }];
            }
          });
        } else if (data.type === 'complete') {
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
        console.error('SSE parse error:', e);
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
    if (!trimmedName) return;

    const newTopic = { name: trimmedName, progress: 0 };

    // Optimistically add to processing list
    setProcessingTopics((prev) => [...prev, newTopic]);

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
        // Rollback if submission fails
        setProcessingTopics((prev) => prev.filter(t => t.name !== trimmedName));
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
      <ul>
        {processingTopics.map((topic, i) => (
          <li key={i}>
            {topic.name} - Progress: {topic.progress}%
          </li>
        ))}
      </ul>

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

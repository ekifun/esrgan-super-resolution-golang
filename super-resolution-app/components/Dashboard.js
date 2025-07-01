import React, { useEffect, useState } from 'react';

function App() {
  const [processedTopics, setProcessedTopics] = useState([]);
  const [processingTopics, setProcessingTopics] = useState([]);
  const [topicName, setTopicName] = useState('');
  const [imageURL, setImageURL] = useState('');

  const fetchTopics = () => {
    fetch('/get-status')
      .then((res) => res.json())
      .then((data) => {
        setProcessedTopics(data.processed || []);
        setProcessingTopics(data.processing || []);
      })
      .catch((err) => {
        console.error('Failed to fetch topics:', err);
      });
  };

  useEffect(() => {
    fetchTopics(); // Initial load
    const interval = setInterval(fetchTopics, 5000); // Refresh every 5s
    return () => clearInterval(interval); // Cleanup
  }, []);

  const handleSubmit = (e) => {
    e.preventDefault();

    fetch('/submit-topic', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ topicName, imageURL }),
    })
      .then((res) => res.json())
      .then((data) => {
        console.log('Submitted successfully:', data);
        setTopicName('');
        setImageURL('');
        fetchTopics(); // Optional immediate refresh
      })
      .catch((err) => console.error('Submission error:', err));
  };

  return (
    <div style={{ padding: '20px', fontFamily: 'sans-serif' }}>
      <h1>Processed Topics</h1>
      <ul>
        {processedTopics.map((topic, i) => (
          <li key={i}>{topic.name}</li>
        ))}
      </ul>

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

export default App;

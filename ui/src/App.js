import './App.css';
import React, { useState, useEffect } from 'react';

function App() {
  const baseUrl = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080';

  const [start, setStart] = useState([]);
  const [counter, setCounter] = useState([]);

  useEffect(() => {
    fetch(baseUrl)
      .then(response => response.json())
      .then(data => {
        setStart(new Date(data.start))
        setCounter(data.counter)
      });
  }, [baseUrl]);

  return (
    <div className="App">
      <p>scottfrazer.net</p>
      <div className="footer">
        {counter} visitors; deployed on {start.toLocaleString()}
      </div>
    </div>
  );
}

export default App;

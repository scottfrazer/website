import './App.css';
import React, { useState, useEffect } from 'react';

function App() {

  console.log(process.env)
  const baseUrl = process.env.API_BASE_URL || 'http://localhost:8080';

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
      <header className="App-header">
        <p>{counter}</p>
        <p>{start.toLocaleString()}</p>
      </header>
    </div>
  );
}

export default App;

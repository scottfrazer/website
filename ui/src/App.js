import './App.css';
import Source from './Source'
import React, { useState, useEffect } from 'react';
import moment from 'moment'

function App() {
  const baseUrl = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080';

  const [start, setStart] = useState([]);
  const [counter, setCounter] = useState([]);
  const [gitHash, setGitHash] = useState([]);
  const [gitHashUrl, setGitHashUrl] = useState([]);

  useEffect(() => {
    fetch(baseUrl)
      .then(response => response.json())
      .then(data => {
        const s = moment(new Date(data.start))
        setStart(s.format('YYYY-MM-DD HH:mm:ss'))
        setCounter(data.counter)
        setGitHash(data.git_hash.substring(0, 8))
        setGitHashUrl("https://github.com/scottfrazer/website/tree/" + data.git_hash)
      });
  }, [baseUrl]);

  return (
    <div className="App">
      <header className="header">
        Scott Frazer
      </header>

      <main className="main">
        <p>👋 Hello!  This is my site.  I'm still 🚧 working on it 🚧</p>
        <Source />
        <p></p>
      </main>

      <footer className="footer">
        {counter} visitors since deployed at {start} (<a href={gitHashUrl}>{gitHash}</a>)
      </footer>
    </div>
  );
}

export default App;

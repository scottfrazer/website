import './App.css';
import Source from './Source'
import React, { useState, useEffect } from 'react';
import moment from 'moment'
import { createBrowserRouter, RouterProvider, useParams, Navigate, useNavigate } from "react-router-dom";

const baseUrl = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8080';

function apiRequest(path, options) {
  options = options || {}
  options.headers = options.headers || {}
  options.headers['Authorization'] = localStorage.getItem("session")
  return fetch(baseUrl + path, options)
}

function Logout(props) {
  localStorage.removeItem("session");
  props.setIsLoggedIn(false)
  return <Navigate to="/" />
}

function Login(props) {
  const navigate = useNavigate()
  const [password, setPassword] = useState([]);

  const save = function (e) {
    e.preventDefault()
    fetch(baseUrl + "/login", { method: "post", body: password })
      .then(response => response.json())
      .then(data => {
        if (data.error) {
          return
        }
        localStorage.setItem("session", data.session);
        props.setIsLoggedIn(true)
        navigate("/")
      });
  }

  return <div>
    <form className="row align-items-center" onSubmit={save}>
      <div className="col-auto">
        <input id="password" type="password" className="form-control" onChange={e => setPassword(e.target.value)} />
      </div>
      <div className="col-auto">
        <input className="btn btn-primary" type="submit" />
      </div>
    </form>
  </div>
}

function App() {
  const [start, setStart] = useState([]);
  const [counter, setCounter] = useState([]);
  const [gitHash, setGitHash] = useState([]);
  const [gitHashUrl, setGitHashUrl] = useState([]);
  const [isLoggedIn, setIsLoggedIn] = useState(false);

  useEffect(() => {
    apiRequest("/")
      .then(response => response.json())
      .then(data => {
        const s = moment(new Date(data.start))
        setStart(s.format('YYYY-MM-DD HH:mm:ss'))
        setCounter(data.counter)
        setGitHash(data.git_hash.substring(0, 8))
        setGitHashUrl("https://github.com/scottfrazer/website/tree/" + data.git_hash)
        setIsLoggedIn(data.logged_in)
      });
  }, []);

  const router = createBrowserRouter([
    {
      path: "/",
      element: <Blog isLoggedIn={isLoggedIn} />,
    },
    {
      path: "/login",
      element: <Login setIsLoggedIn={setIsLoggedIn} />,
    },
    {
      path: "/edit/:id",
      element: <Edit />,
    },
    {
      path: "/create",
      element: <Edit />,
    },
    {
      path: "/logout",
      element: <Logout setIsLoggedIn={setIsLoggedIn} />,
    },
  ]);

  return (
    <div className="App">
      <header className="header">
        <div className="title">
          <a href="/">Scott Frazer</a>
        </div>
        <div className="nav">
          {!isLoggedIn && <a href="/login" type="button">🔑</a>}
          {isLoggedIn && <a href="/create" type="button">✏️</a>}
          {isLoggedIn && <a href="/logout" type="button">❌</a>}
        </div>
      </header>

      <main className="main">
        <RouterProvider router={router} />
      </main>

      <footer className="footer">
        {counter} visitors since deployed at {start} (<a href={gitHashUrl}>{gitHash}</a>)
      </footer>
    </div >
  );
}

function Edit() {
  const { id } = useParams();
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [date, setDate] = useState("");
  const [error, setError] = useState(undefined);
  const [status, setStatus] = useState();


  useEffect(() => {
    if (!id) {
      return
    }

    apiRequest(`/blog/id/${id}`)
      .then(response => response.json())
      .then(data => {
        if (data.error) {
          setError(data.error)
        } else {
          setError(undefined)
          setTitle(data.title)
          setContent(data.content)
          setDate(moment.utc(data.date, 'YYYY-MM-DDTHH:mm:ssZ').format('YYYY-MM-DDTHH:mm'))
        }
      })
  }, [id])

  var save = function (e) {
    e.preventDefault()

    const options = {
      method: "post",
      body: JSON.stringify({
        id: parseInt(id),
        title: title,
        date: moment(date, 'YYYY-MM-DDTHH:mm').utcOffset(0).format('YYYY-MM-DDTHH:mm:ssZ'),
        content: content
      })
    }

    var url = "/blog"
    if (id) {
      url = `/blog/id/${id}`
    }

    apiRequest(url, options)
      .then(response => response.json())
      .then(data => {
        if (data.error) {
          setStatus(<div className="mt-3 alert alert-danger" role="alert">Error: {data.error}...</div>)
          return
        }

        setTitle(data.title)
        setContent(data.content)
        setDate(moment.utc(data.date, 'YYYY-MM-DDTHH:mm:ssZ').format('YYYY-MM-DDTHH:mm'))

        const s = 3
        var c = 0
        const status = function () {
          setStatus(<div className="mt-3 alert alert-success" role="alert">Success! {s - c}...</div>)
        }

        status()
        const intervalId = setInterval(function () {
          c++
          status()
        }, 1000);

        setTimeout(() => {
          clearInterval(intervalId);
          setStatus(<div />)
        }, s * 1000);
      })
      .catch(e => setStatus(<div className="mt-3 alert alert-danger" role="alert">Error: {e}...</div>))
  }

  if (error !== undefined) {
    return <div className="error">{error}</div>
  }

  return <div className="editor">
    {status}
    <form onSubmit={save}>
      <div className="mb-3">
        <label htmlFor="title" className="form-label">Title</label>
        <input id="title" type="text" className="form-control" value={title} onChange={e => setTitle(e.target.value)} />
      </div>
      <div className="mb-3">
        <label htmlFor="date" className="form-label">Date</label>
        <input id="date" type="datetime-local" className="form-control" value={date} onChange={e => setDate(e.target.value)} />
      </div>
      <div className="mb-3">
        <label htmlFor="content" className="form-label">Content</label>
        <textarea id="content" className="form-control" value={content} onChange={e => setContent(e.target.value)} />
      </div>
      <div className="mb-3">
        <input className="btn btn-primary" type="submit" />
      </div>
    </form>
  </div>
}

function Blog(props) {
  const [blogPost, setBlogPost] = useState([]);
  const [error, setError] = useState(undefined);

  useEffect(() => {
    apiRequest(`/blog/latest`)
      .then(response => response.json())
      .then(data => {
        if (data.error) {
          setError(data.error)
        } else {
          setBlogPost(data)
        }
      })
      .catch(e => setError(`could not fetch latest entry: ${e.message}`))
  }, [])

  if (error) {
    return <div className="error">{error}</div>
  }

  return (
    <div className="blog">
      <h1>{blogPost.title}</h1>
      <div className="blog-post-meta">{blogPost.date}</div>
      {paragraphs(blogPost)}
      {props.isLoggedIn && blogPost.id && <a href={`/edit/${blogPost.id}`}>✏️</a>}
    </div>
  )
}

function paragraphs(data) {
  if (!data || !data.content) {
    return []
  }
  const codeRegex = /^\{code\s+language="(.*)"\}/;
  const codeEndRegex = /^\{code\}/;
  const newlineRegex = /^\n{2,}/
  var content = ''
  var language = ''
  var blockType = 'p' // p for paragraph, c for code
  var blocks = []
  var key = 0

  const flush = function () {
    content = content.trim()
    if (content.length > 0) {
      if (blockType === 'p') {
        blocks.push(<Paragraph key={key} content={content} />)
      }
      if (blockType === 'c') {
        blocks.push(<Source key={key} code={content} language={language} />)
      }
      key++
      content = ''
      blockType = 'p'
    }
  }

  var s = data.content
  while (s.length > 0) {
    let m = codeRegex.exec(s)
    if (m) {
      flush()
      language = m[1]
      blockType = 'c'
      s = s.substring(m[0].length)
      continue
    }

    m = newlineRegex.exec(s)
    if (m && blockType === 'p') {
      flush()
      s = s.substring(m[0].length)
      continue
    }

    m = codeEndRegex.exec(s)
    if (m && blockType === 'c') {
      flush()
      s = s.substring(m[0].length)
      continue
    }

    content = content + s[0]
    s = s.substring(1)
  }

  flush()

  return blocks
}

function Paragraph(props) {
  return <p>{props.content}</p>
}

export default App;

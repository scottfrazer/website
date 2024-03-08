import './App.css';
import Source from './Source'
import React, { useState, useEffect } from 'react';
import moment from 'moment'
import { Routes, Route, BrowserRouter, useParams, Navigate, useNavigate, Link, useLocation } from "react-router-dom";
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faCoffee } from '@fortawesome/free-solid-svg-icons'


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

  return <div className="login">
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

  return (
    <BrowserRouter>
      <div className="App">
        <header className="header">
          <div className="title">
            <Link to="/">Scott Frazer</Link>
          </div>
          <div className="nav">
            <Link to="/list">🗄</Link>
            <Link to="/running">🏃</Link>
            {!isLoggedIn && <Link to="/login">🔑</Link>}
            {isLoggedIn && <Link to="/create">✏️</Link>}
            {isLoggedIn && <Link to="/logout">❌</Link>}
          </div>
        </header>

        <main className="main">
          <Routes>
            <Route path="/" element=<Blog isLoggedIn={isLoggedIn} /> />
            <Route path="/running" element=<Running page={1} perPage={50} /> />
            <Route path="/list" element=<List /> />
            <Route path="/edit/:id" element=<Edit /> />
            <Route path="/create" element=<Edit /> />
            <Route path="/login" element=<Login setIsLoggedIn={setIsLoggedIn} /> />
            <Route path="/logout" element=<Logout setIsLoggedIn={setIsLoggedIn} /> />
          </Routes>
        </main>

        <footer className="footer">
          {counter} visits since deployed at {start} (<a href={gitHashUrl}>{gitHash}</a>)
        </footer>
      </div>
    </BrowserRouter>
  );
}

function Running(props) {
  const [activities, setActivities] = useState([]);
  const [error, setError] = useState([]);

  const { search } = useLocation();
  const queryParams = new URLSearchParams(search);
  var { page: pageFromQuery, perPage: perPageFromQuery } = Object.fromEntries(queryParams.entries());

  const isNumeric = (value) => !isNaN(value);

  if (!isNumeric(pageFromQuery)) {
    pageFromQuery = 1
  } else {
    pageFromQuery = parseInt(pageFromQuery)
  }

  if (!isNumeric(perPageFromQuery)) {
    perPageFromQuery = 50
  } else {
    perPageFromQuery = parseInt(perPageFromQuery)
  }

  useEffect(() => {
    apiRequest(`/running/list?page=${pageFromQuery}&perPage=${perPageFromQuery}`)
      .then(response => response.json())
      .then(data => {
        if (data.error) {
          setError(data.error)
        } else {
          data = data.map(p => {
            const s = moment(p.date).utc()
            p.date = s.format('YYYY-MM-DD')
            p.time = s.format('hh:mm')
            return p
          })
          setError(undefined)
          setActivities(data)
        }
      })
  }, [pageFromQuery, perPageFromQuery])

  if (error !== undefined) {
    return <div>Error: {error}</div>
  }

  console.log(pageFromQuery)
  console.log(perPageFromQuery)
  console.log(activities.length)

  return <div className="running">
    <div>
      {pageFromQuery === 1 ? <span>prev</span> : <Link to={{ pathname: '/running', search: `?page=${pageFromQuery - 1}&perPage=${perPageFromQuery}` }}>prev</Link>}
      &nbsp;|&nbsp;
      {activities.length !== perPageFromQuery ? <span>next</span> : <Link to={{ pathname: '/running', search: `?page=${pageFromQuery + 1}&perPage=${perPageFromQuery}` }}>next</Link>}
    </div>
    <table>
      <thead>
        <tr>
          <td>Date</td>
          <td>Time</td>
          <td>Title</td>
          <td>Distance</td>
          <td>Type</td>
          <td>Pace</td>
          <td>Time</td>
          <td>Link</td>
        </tr>
      </thead>
      <tbody>
        {activities.map((activity) => (
          <tr key={activity.id}>
            <td>{activity.date}</td>
            <td>{activity.time}</td>
            <td class="title">{activity.title}</td>
            <td>{activity.distance}</td>
            <td>{activity.type}</td>
            <td>{activity.pace}/mi</td>
            <td>{activity.moving_time}</td>
            <td><a href={`https://strava.com/activities/${activity.id}`}><FontAwesomeIcon icon={faCoffee} /></a></td>
          </tr>
        ))}
      </tbody>
    </table>
  </div >
}

function List() {
  const [posts, setPosts] = useState([]);
  const [error, setError] = useState([]);

  useEffect(() => {
    apiRequest(`/blog/list`)
      .then(response => response.json())
      .then(data => {
        if (data.error) {
          setError(data.error)
        } else {
          setError(undefined)
          data = data.map(p => {
            const s = moment(new Date(p.date))
            p.date = s.format('YYYY-MM-DD')
            return p
          })
          setPosts(data)
        }
      })
  }, [])

  if (error !== undefined) {
    return <div>Error: {error}</div>
  }

  return <div className="bloglist">
    <ul>
      {posts.map((post) => (<li key={post.id}>{post.date} - <Link to={`/edit/${post.id}`}>{post.title}</Link></li>))}
    </ul>
  </div>
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
  const [displayDate, setDisplayDate] = useState("")
  const [error, setError] = useState(undefined);

  useEffect(() => {
    apiRequest(`/blog/latest`)
      .then(response => response.json())
      .then(data => {
        if (data.error) {
          setError(data.error)
        } else {
          const s = moment(new Date(data.date))
          setDisplayDate(s.format('YYYY-MM-DD'))
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
      <div className="mb-3">📅&nbsp;&nbsp;{displayDate}</div>
      {paragraphs(blogPost)}
      {props.isLoggedIn && blogPost.id && <Link to={`/edit/${blogPost.id}`}>✏️</Link>}
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

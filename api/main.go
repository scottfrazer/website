package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/scottfrazer/running/strava"

	"golang.org/x/crypto/bcrypt"
)

var GitHash string

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type BlogRepo struct {
	db *sql.DB
}

type BlogPost struct {
	Id      int64     `json:"id"`
	Title   string    `json:"title"`
	Date    time.Time `json:"date"`
	Content string    `json:"content"`
}

func NewBlogRepo(db *sql.DB) *BlogRepo {
	r := &BlogRepo{db}
	check(r.Init())
	return r
}

func (repo BlogRepo) List() ([]BlogPost, error) {
	rows, err := repo.db.Query("SELECT id, title, date, content FROM blog;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []BlogPost{}

	for rows.Next() {
		var id int64
		var title string
		var date time.Time
		var content string
		if err := rows.Scan(&id, &title, &date, &content); err != nil {
			return nil, err
		}
		result = append(result, BlogPost{id, title, date, content})
	}
	return result, nil
}

func (repo BlogRepo) Init() error {
	_, err := repo.db.Exec("CREATE TABLE IF NOT EXISTS blog (id bigserial, title text, date timestamp, content text)")
	return err
}

func (repo BlogRepo) Get(id int64) (*BlogPost, error) {
	var title string
	var date time.Time
	var content string
	err := repo.db.QueryRow("SELECT title, date, content FROM blog WHERE id=$1", id).Scan(&title, &date, &content)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &BlogPost{id, title, date, content}, nil
}

func (repo BlogRepo) GetLatest() (*BlogPost, error) {
	var id int64
	var title string
	var date time.Time
	var content string
	err := repo.db.QueryRow("SELECT id, title, date, content FROM blog ORDER BY date DESC LIMIT 1").Scan(&id, &title, &date, &content)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &BlogPost{id, title, date, content}, nil
}

func (repo BlogRepo) Set(blogPost BlogPost) error {
	_, err := repo.db.Exec(
		"UPDATE blog SET title=$1, date=$2, content=$3 WHERE id=$4",
		blogPost.Title,
		blogPost.Date.UTC(),
		blogPost.Content,
		blogPost.Id,
	)
	return err
}
func (repo BlogRepo) Create(title string, date time.Time, content string) (BlogPost, error) {
	row := repo.db.QueryRow(
		"INSERT INTO blog (title, date, content) VALUES ($1, $2, $3) RETURNING id",
		title, date, content,
	)

	var id int64
	if err := row.Scan(&id); err != nil {
		return BlogPost{}, err
	}

	return BlogPost{id, title, date, content}, nil
}

var originAllowlist = []string{
	"http://127.0.0.1:3000",
	"http://localhost:3000",
	"https://scottfrazer.net",
}

func isAllowed(origin string) bool {
	for _, allowed := range originAllowlist {
		if origin == allowed {
			return true
		}
	}
	return false
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

func setLoggedIn(ctx context.Context) context.Context {
	return context.WithValue(ctx, "logged_in", true)
}

func isLoggedIn(ctx context.Context) bool {
	return ctx.Value("logged_in") == true
}

func session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := tokens[r.Header.Get("Authorization")]; ok {
			r = r.WithContext(setLoggedIn(r.Context()))
		}
		next.ServeHTTP(w, r)
	})
}

func admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session := r.Header.Get("Authorization")
		if _, ok := tokens[session]; !ok {
			w.WriteHeader(403)
			fmt.Fprintf(w, `{"error": "not logged in"}`)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var tokens map[string]string

func main() {
	ctx := context.Background()
	db, err := sql.Open("postgres", os.Getenv("POSTGRES_DSN"))
	check(err)

	blogRepo := NewBlogRepo(db)

	tokens = map[string]string{}

	c := 0
	s := time.Now()
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors)
	r.Use(session)

	store, err := strava.NewPostgresDataStore(os.Getenv("POSTGRES_DSN"))
	check(err)
	session, err := store.GetSession()
	check(err)
	if session == nil {
		session = &strava.StravaSession{
			RefreshToken: os.Getenv("STRAVA_REFRESH_TOKEN"),
			ClientId:     os.Getenv("STRAVA_CLIENT_ID"),
			ClientSecret: os.Getenv("STRAVA_SECRET_KEY"),
		}
	}
	stravaClient, err := strava.NewStravaClientFromSession(*session)
	check(err)
	check(stravaClient.Sync(ctx, store))

	r.Method("OPTIONS", "/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers for CORS preflight requests
		w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Respond with a 200 status code to indicate that CORS is allowed
		w.WriteHeader(http.StatusOK)
	}))
	r.Get("/running/stats", func(w http.ResponseWriter, r *http.Request) {
		all, err := store.Load(strava.ActivityFilter{})
		check(err)

		type UiStats struct {
			MilesPerYear     map[int]float64 `json:"miles_per_year"`
			ActivitesPerYear map[int]int     `json:"activites_per_year"`
		}
		stats := UiStats{
			MilesPerYear:     map[int]float64{},
			ActivitesPerYear: map[int]int{},
		}
		for _, activity := range all {
			year := activity.Date().Year()
			stats.MilesPerYear[year] += activity.Miles()
			stats.ActivitesPerYear[year] += 1
		}

		bytes, err := json.Marshal(stats)
		check(err)
		_, err = w.Write(bytes)
		check(err)
	})
	r.Get("/running/list", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		queryPage := query.Get("page")
		queryPerPage := query.Get("perPage")

		page, _ := strconv.ParseInt(queryPage, 10, 64)
		perPage, _ := strconv.ParseInt(queryPerPage, 10, 64)

		if page == 0 {
			page = 1
		}
		if perPage == 0 {
			perPage = 50
		}

		type UiActivity struct {
			Id          int64  `json:"id"`
			Title       string `json:"title"`
			MovingTime  string `json:"moving_time"`
			Pace        string `json:"pace"`
			Distance    string `json:"distance"`
			Type        string `json:"type"`
			WorkoutType int    `json:"workout_type"`
			Date        string `json:"date"`
		}

		activities, err := store.LoadPage(int(page), int(perPage))
		check(err)

		uiActivities := []UiActivity{}
		for _, activity := range activities {
			uiActivities = append(uiActivities, UiActivity{
				Id:          activity.Id,
				Title:       activity.Name,
				MovingTime:  activity.MovingTimeString(),
				Pace:        activity.PacePerMile(),
				Distance:    activity.DistanceString(),
				Type:        activity.Type,
				WorkoutType: activity.WorkoutType,
				Date:        activity.Date().Format(time.RFC3339),
			})
		}

		bytes, err := json.Marshal(uiActivities)
		check(err)
		_, err = w.Write(bytes)
		check(err)
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		c++
		b, err := json.Marshal(struct {
			Counter  int       `json:"counter"`
			Start    time.Time `json:"start"`
			GitHash  string    `json:"git_hash"`
			LoggedIn bool      `json:"logged_in"`
		}{
			Counter:  c,
			Start:    s,
			GitHash:  GitHash,
			LoggedIn: isLoggedIn(r.Context()),
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(b)
	})
	r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		check(err)
		if bcrypt.CompareHashAndPassword([]byte(os.Getenv("ADMIN_PASSWORD_BCRYPT")), body) == nil {
			token := uuid.NewString()
			tokens[token] = ""
			fmt.Fprintf(w, `{"session": "%s"}`, token)
			return
		}
		fmt.Fprintf(w, `{"error": "invalid password"}`)
		w.WriteHeader(http.StatusForbidden)
	})
	r.With(admin).Post("/blog", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		check(err)
		var post BlogPost
		check(json.Unmarshal(body, &post))
		post, err = blogRepo.Create(post.Title, post.Date, post.Content)
		check(err)
		b, err := json.Marshal(post)
		check(err)
		w.Write(b)
	})
	r.Get("/blog/latest", func(w http.ResponseWriter, r *http.Request) {
		post, err := blogRepo.GetLatest()
		check(err)
		if post == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, `{"error": "post not found"}`)
			return
		}
		b, err := json.Marshal(post)
		check(err)
		w.Write(b)
	})
	r.Get("/blog/id/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		check(err)
		post, err := blogRepo.Get(id)
		check(err)
		if post == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, `{"error": "post not found"}`)
			return
		}
		b, err := json.Marshal(post)
		check(err)
		w.Write(b)
	})
	r.With(admin).Post("/blog/id/{id}", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		check(err)
		var post BlogPost
		check(json.Unmarshal(body, &post))
		check(blogRepo.Set(post))
		w.Write(body)
	})
	r.Get("/blog/list", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		list, err := blogRepo.List()
		check(err)
		listBytes, err := json.Marshal(list)
		check(err)
		w.Write(listBytes)
	})
	port := "0.0.0.0:8080"
	fmt.Printf("listening on %s...\n", port)
	http.ListenAndServe(port, r)
}

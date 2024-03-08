package strava

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/time/rate"

	_ "github.com/lib/pq"
)

type SummaryActivity struct {
	Id          int64         `json:"id"`
	Name        string        `json:"name"`
	DateString  string        `json:"start_date_local"`
	Distance    float64       `json:"distance"`
	MovingTime  float64       `json:"moving_time"`
	WorkoutType int           `json:"workout_type"`
	Type        string        `json:"type"`
	Map         ActivityMap   `json:"map"`
	Laps        []ActivityLap `json:"laps"`
}

type ActivityLap struct {
	Id                 int64     `json:"id"`
	ResourceState      int32     `json:"resource_state"`
	Name               string    `json:"name"`
	ElapsedTime        int32     `json:"elapsed_time"`
	MovingTime         int32     `json:"moving_time"`
	StartDate          time.Time `json:"start_date"`
	StartDateLocal     time.Time `json:"start_date_local"`
	Distance           float64   `json:"distance"`
	StartIndex         int32     `json:"start_index"`
	EndIndex           int32     `json:"end_index"`
	TotalElevationGain float64   `json:"total_elevation_gain"`
	AverageSpeed       float64   `json:"average_speed"`
	MaxSpeed           float64   `json:"max_speed"`
	AverageCadence     float64   `json:"average_cadence"`
	DeviceWatts        bool      `json:"device_watts"`
	AverageWats        float64   `json:"average_watts"`
	LapIndex           int32     `json:"lap_index"`
	Split              int32     `json:"split"`
}

type SummaryActivityDateSort []SummaryActivity

func (tds SummaryActivityDateSort) Len() int {
	return len(tds)
}

func (tds SummaryActivityDateSort) Swap(i, j int) {
	tds[i], tds[j] = tds[j], tds[i]
}

func (tds SummaryActivityDateSort) Less(i, j int) bool {
	return tds[i].Date().Before(tds[j].Date())
}

type ActivityMap struct {
	Id            string `json:"id"`
	ResourceState int    `json:"resource_state"`
	Polyline      string `json:"summary_polyline"`
}

func (a *SummaryActivity) Date() time.Time {
	// TODO: ignoring error
	t, _ := time.Parse("2006-01-02T15:04:05Z", a.DateString)
	return t
}

func (a *SummaryActivity) IsRace() bool {
	return a.WorkoutType == 1
}

func (a *SummaryActivity) Miles() float64 {
	return (a.Distance / 1000) * 0.621371
}

func (a *SummaryActivity) DistanceString() string {
	return fmt.Sprintf("%s mi", strconv.FormatFloat(a.Miles(), 'f', 2, 64))
}

func (a *SummaryActivity) MovingTimeString() string {
	d := time.Duration(a.MovingTime) * time.Second
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func (a *SummaryActivity) PacePerMile() string {
	d := time.Duration(a.MovingTime/a.Miles()) * time.Second
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02d:%02d", m, s)
}

type StravaAthlete struct {
	Id            int64     `json:"id"`
	Username      string    `json:"username"`
	ResourceState int       `json:"resource_state"`
	FirstName     string    `json:"firstname"`
	LastName      string    `json:"lastname"`
	City          string    `json:"city"`
	State         string    `json:"state"`
	Country       string    `json:"country"`
	Sex           string    `json:"sex"`
	Premium       bool      `json:"premium"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type StravaSession struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    int64  `json:"expires_at"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

func (session *StravaSession) Save() error {
	usr, _ := user.Current()
	root := path.Join(usr.HomeDir, ".gorun")
	err := createDirectoryIfNotExists(root)
	if err != nil {
		return err
	}
	sessionPath := path.Join(root, "strava_session.json")

	jsonString, err := session.Json()
	if err != nil {
		return err
	}

	err = os.WriteFile(sessionPath, []byte(jsonString), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (session StravaSession) Json() (string, error) {
	json, err := json.MarshalIndent(session, "", "    ")
	if err != nil {
		return "", err
	}
	return string(json), nil
}

func (session StravaSession) IsExpired() bool {
	expiresAt := time.Unix(session.ExpiresAt, 0)
	now := time.Now().UTC()
	return now.After(expiresAt)
}

type StravaClient struct {
	session *StravaSession
	limiter *rate.Limiter
}

func NewStravaClient() (*StravaClient, error) {
	return nil, nil
}

func refreshSession(session *StravaSession) error {
	body, err := json.Marshal(map[string]string{
		"client_id":     session.ClientId,
		"client_secret": session.ClientSecret,
		"grant_type":    "refresh_token",
		"refresh_token": session.RefreshToken,
	})

	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://www.strava.com/api/v3/oauth/token",
		bytes.NewReader(body),
	)

	if err != nil {
		return err
	}

	req.Header.Add("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	jsonBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var newSession StravaSession
	if err := json.Unmarshal(jsonBytes, &newSession); err != nil {
		return err
	}

	newSession.ClientId = session.ClientId
	newSession.ClientSecret = session.ClientSecret
	*session = newSession
	return nil
}

func NewStravaClientFromSession(session StravaSession) (*StravaClient, error) {
	if session.IsExpired() {
		if err := refreshSession(&session); err != nil {
			return nil, err
		}
	}

	return &StravaClient{
		session: &session,
		limiter: rate.NewLimiter(100.0/(60*15), 10),
	}, nil
}

func NewStravaClientFromBrowserBasedLogin(clientId, clientSecret string, store DataStore) (*StravaClient, error) {
	port := 9753
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)

	signal.Notify(quit, os.Interrupt)
	defer signal.Reset(os.Interrupt)

	router := http.NewServeMux()
	router.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		session, err := apiGetSessionFromAuthorizationCode(clientId, clientSecret, r.URL.Query()["code"][0])
		if err != nil {
			panic(err)
		}

		fmt.Printf("/callback: %+v\n", session)
		if err := store.SaveSession(session); err != nil {
			panic(err)
		}

		fmt.Fprintf(w, "<p>Authentication complete.  This browser window can be closed</p>")
		quit <- os.Interrupt
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// Wait for OAuth callback to be hit, then shutdown HTTP server
	go func(server *http.Server, quit <-chan os.Signal, done chan<- bool) {
		<-quit
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("error: %v", err)
		}
		close(done)
	}(server, quit, done)

	// Start HTTP server waiting for OAuth callback
	go func(server *http.Server) {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("error: %v", err)
		}
	}(server)

	// Open this URL to start the process of authentication
	authorizeURL, err := url.Parse("https://strava.com/oauth/authorize")
	if err != nil {
		return nil, err
	}
	q := authorizeURL.Query()
	q.Add("client_id", clientId)
	q.Add("response_type", "code")
	q.Add("redirect_uri", fmt.Sprintf("http://localhost:%d/callback", port))
	q.Add("scope", "read_all")
	q.Add("scope", "activity:read_all")
	q.Add("approval_prompt", "force")
	authorizeURL.RawQuery = q.Encode()

	// Open web browser to authorize
	fmt.Printf("Opening web browser to initiate authentication...\n")
	open(authorizeURL.String())

	// Wait for exchange to finish
	<-done

	session, err := store.GetSession()
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("unexpected error: no session found")
	}
	return NewStravaClientFromSession(*session)
}

func (c *StravaClient) httpReq(method string, url string, headers map[string]string, body []byte, expectedStatus int) (*http.Response, error) {
	start := time.Now()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if _, ok := headers["Authorization"]; !ok && c.session.AccessToken != "" {
		headers["Authorization"] = "Bearer " + c.session.AccessToken
	}

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(responseBody))

	log.Printf("%s %s ... %s (%s; %s)", req.Method, req.URL.String(), resp.Status, time.Since(start), humanize.Bytes(uint64(len(responseBody))))

	if expectedStatus != -1 && resp.StatusCode != expectedStatus {
		return resp, fmt.Errorf("unexpected status: %d (expected %d)", resp.StatusCode, expectedStatus)
	}

	return resp, nil
}

func apiGetSessionFromAuthorizationCode(clientId, clientSecret, authorizationCode string) (*StravaSession, error) {
	body, err := json.Marshal(map[string]string{
		"client_id":     clientId,
		"client_secret": clientSecret,
		"grant_type":    "authorization_code",
		"code":          authorizationCode,
	})

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(
		http.MethodPost,
		"https://www.strava.com/api/v3/oauth/token",
		bytes.NewReader(body),
	)

	if err != nil {
		return nil, err
	}

	req.Header.Add("content-type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	jsonBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var newsession StravaSession
	if err := json.Unmarshal(jsonBytes, &newsession); err != nil {
		return nil, err
	}

	newsession.ClientId = clientId
	newsession.ClientSecret = clientSecret

	return &newsession, nil
}

func (c *StravaClient) apiGetActivities(ctx context.Context, page int, mostRecent time.Time) ([]SummaryActivity, error) {
	c.limiter.Wait(ctx)

	url := fmt.Sprintf("https://www.strava.com/api/v3/athlete/activities?page=%d&per_page=200", page)
	if !mostRecent.IsZero() {
		url = url + fmt.Sprintf("&after=%d", mostRecent.Unix())
	}

	resp, err := c.httpReq(
		"GET",
		url,
		map[string]string{},
		[]byte{},
		-1,
	)
	log.Printf("GET %s: %d", url, resp.StatusCode)

	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var activities []SummaryActivity
	err = json.Unmarshal(body, &activities)
	if err != nil {
		return nil, err
	}
	return activities, nil
}

func (c *StravaClient) apiGetAthlete(ctx context.Context) (*StravaAthlete, error) {
	c.limiter.Wait(ctx)

	resp, err := c.httpReq(
		"GET",
		"https://www.strava.com/api/v3/athlete",
		map[string]string{},
		[]byte{},
		-1,
	)

	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var athlete StravaAthlete
	err = json.Unmarshal(body, &athlete)
	if err != nil {
		return nil, err
	}
	return &athlete, nil
}

func (c *StravaClient) apiGetLaps(ctx context.Context, activityId int64) ([]ActivityLap, error) {
	c.limiter.Wait(ctx)

	url := fmt.Sprintf("https://www.strava.com/api/v3/activities/%d/laps", activityId)
	resp, err := c.httpReq(
		"GET",
		url,
		map[string]string{},
		[]byte{},
		-1,
	)
	log.Printf("GET %s: %d", url, resp.StatusCode)

	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var laps []ActivityLap
	err = json.Unmarshal(body, &laps)
	if err != nil {
		return nil, err
	}
	return laps, nil
}

func (c *StravaClient) Sync(ctx context.Context, store DataStore) error {
	mostRecent, err := store.GetMostRecentActivityDate()
	if err != nil {
		return err
	}

	for i := 1; ; i++ {
		activities, err := c.apiGetActivities(ctx, i, mostRecent)
		if err != nil {
			return err
		}
		if len(activities) == 0 {
			break
		}
		for _, activity := range activities {
			laps, err := c.apiGetLaps(ctx, activity.Id)
			if err != nil {
				return err
			}
			if err := store.SaveLaps(activity.Id, laps); err != nil {
				return err
			}
		}
		if err := store.Save(activities); err != nil {
			return err
		}
	}

	return nil
}

// open opens the specified URL in the default browser of the user.
func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func createDirectoryIfNotExists(dirName string) error {
	src, err := os.Stat(dirName)

	if os.IsNotExist(err) {
		err := os.MkdirAll(dirName, 0755)
		if err != nil {
			return err
		}
		return nil
	}

	if src.Mode().IsRegular() {
		return fmt.Errorf("directory %s is a regular file", dirName)
	}

	return nil
}

type DataStore struct {
	db *sql.DB
}

func NewPostgresDataStore(dsn string) (DataStore, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return DataStore{}, err
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS strava_activities (
			id bigserial primary key,
			value jsonb
		)`,

		`CREATE TABLE IF NOT EXISTS strava_activities (
			id bigserial primary key,
			start_date timestamptz,
			value jsonb
		)`,

		`CREATE TABLE IF NOT EXISTS strava_laps (
			id bigserial primary key,
			activity_id text,
			value jsonb
		)`,

		`CREATE INDEX IF NOT EXISTS strava_activities_date ON strava_activities (start_date DESC)`,
	}

	for _, query := range queries {
		_, err = db.Exec(query)
		if err != nil {
			return DataStore{}, err
		}
	}

	return DataStore{db}, nil
}

type ActivityFilter struct {
	start *time.Time
	end   *time.Time
}

func (s *DataStore) GetMostRecentActivityDate() (time.Time, error) {
	query := `SELECT coalesce(max((value->>'start_date_local')::timestamptz), '1970-01-01T00:00:00Z'::timestamptz) FROM strava_activities`
	var t time.Time
	err := s.db.QueryRow(query).Scan(&t)
	return t, err
}

func (s *DataStore) GetSession() (*StravaSession, error) {
	var bytes []byte
	err := s.db.QueryRow("SELECT value FROM strava_session WHERE id=1").Scan(&bytes)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	var session StravaSession
	if err := json.Unmarshal(bytes, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *DataStore) SaveSession(session *StravaSession) error {
	bytes, err := json.Marshal(session)
	if err != nil {
		return err
	}

	query := `INSERT INTO strava_session (id, value)
		VALUES (1, $1)
		ON CONFLICT (id)
		DO UPDATE SET value = EXCLUDED.value`
	if _, err := s.db.Exec(query, bytes); err != nil {
		return err
	}

	return nil
}

func (s *DataStore) Save(activities []SummaryActivity) error {
	query := `INSERT INTO strava_activities (id, start_date, value) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING`
	for _, activity := range activities {
		serialized, err := json.Marshal(activity)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(query, activity.Id, activity.Date(), serialized); err != nil {
			return err
		}
	}
	return nil
}

func (s *DataStore) SaveLaps(activityId int64, laps []ActivityLap) error {
	query := `INSERT INTO strava_laps (id, activity_id, value) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING`
	for _, lap := range laps {
		serialized, err := json.Marshal(lap)
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(query, lap.Id, activityId, serialized); err != nil {
			return err
		}
	}
	return nil
}

func (s *DataStore) activityQuery(query string) ([]SummaryActivity, error) {
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []SummaryActivity{}
	for rows.Next() {
		var value []byte
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		var activity SummaryActivity
		if err := json.Unmarshal(value, &activity); err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}
	return activities, nil
}

func (s *DataStore) LoadPage(page, perPage int) ([]SummaryActivity, error) {
	return s.activityQuery(
		fmt.Sprintf(`
			SELECT value
			FROM strava_activities
			ORDER BY (value->>'start_date_local')::timestamptz DESC
			LIMIT %d
			OFFSET %d`,
			perPage,
			(page-1)*perPage,
		),
	)
}

func (s *DataStore) Load(filters ActivityFilter) ([]SummaryActivity, error) {
	where := []string{}

	if filters.start != nil {
		where = append(
			where,
			fmt.Sprintf(
				"(value->>'start_date_local')::timestamptz >= '%s'::timestamptz",
				filters.start.Format(time.RFC3339),
			),
		)
	}

	if filters.end != nil {
		where = append(
			where,
			fmt.Sprintf(
				"(value->>'start_date_local')::timestamptz < '%s'::timestamptz",
				filters.end.Format(time.RFC3339),
			),
		)
	}

	query := "SELECT value FROM strava_activities"
	if len(where) > 0 {
		query += fmt.Sprintf(" WHERE %s", strings.Join(where, " AND "))
	}
	query += " ORDER BY (value->>'start_date_local')::timestamptz DESC"

	return s.activityQuery(query)
}

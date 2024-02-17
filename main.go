package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type tuple struct {
	id   int64
	data string
}

func tuples(conn string) []tuple {
	db, err := sql.Open("postgres", conn)
	check(err)

	rows, err := db.Query("SELECT id, data FROM test_table")
	check(err)
	defer rows.Close()

	result := []tuple{}
	for rows.Next() {
		var id int64
		var data string
		check(rows.Scan(&id, &data))
		result = append(result, tuple{id, data})
	}
	return result
}

var originAllowlist = []string{
	"http://127.0.0.1:3000",
	"http://localhost:3000",
}

func checkCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if slices.Contains(originAllowlist, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	dsn := os.Getenv("POSTGRES_DSN")

	writeTuples := func(w io.Writer) {
		if dsn != "" {
			for _, t := range tuples(dsn) {
				fmt.Fprintf(w, "(%d, %s)\n", t.id, t.data)
			}
		} else {
			fmt.Fprintf(w, "no connection string\n")
		}
	}

	writeTuples(os.Stdout)

	c := 0
	s := time.Now()
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(checkCORS)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		c++
		b, err := json.Marshal(struct {
			Counter int       `json:"counter"`
			Start   time.Time `json:"start"`
		}{
			Counter: c,
			Start:   s,
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(b)
	})
	r.Get("/tuples", func(w http.ResponseWriter, r *http.Request) {
		writeTuples(w)
	})
	port := "0.0.0.0:8080"
	fmt.Printf("listening on %s...\n", port)
	http.ListenAndServe(port, r)
}

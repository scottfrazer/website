package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"

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

func main() {
	dsn := os.Getenv("POSTGRES")

	writeTuples := func(w io.Writer) {
		if dsn != "" {
			fmt.Printf("connection string: %s\n", dsn)
			for _, t := range tuples(os.Args[1]) {
				fmt.Printf("(%d, %s)\n", t.id, t.data)
			}
		} else {
			fmt.Printf("no connection string\n")
		}
	}

	writeTuples(os.Stdout)

	c := 0
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		c++
		fmt.Fprintf(w, "%d", c)
	})
	r.Get("/tuples", func(w http.ResponseWriter, r *http.Request) {
		writeTuples(w)
	})
	port := "0.0.0.0:8080"
	fmt.Printf("listening on %s...\n", port)
	http.ListenAndServe(port, r)
}

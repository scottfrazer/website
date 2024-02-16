package main

import (
	"database/sql"
	"fmt"
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

func main() {

	if len(os.Args) > 1 && os.Args[1] != "" {
		db, err := sql.Open("postgres", os.Args[1])
		check(err)

		rows, err := db.Query("SELECT id, data FROM test_table")
		check(err)
		for rows.Next() {
			var id int64
			var data string
			check(rows.Scan(&id, &data))
			fmt.Printf("(%d, %s)\n", id, data)
		}
		rows.Close()
	}

	c := 0
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		c++
		fmt.Fprintf(w, "%d", c)
	})
	port := "0.0.0.0:8080"
	fmt.Printf("listening on %s...\n", port)
	http.ListenAndServe(port, r)
}

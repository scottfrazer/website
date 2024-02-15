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

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	http.ListenAndServe(":3000", r)
}

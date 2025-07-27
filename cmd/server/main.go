package main

import (
	"encoding/json"
	"file-archiver/internal/task"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	addr := ":" + getenv("PORT", "8080")

	store := task.NewMemoryStore()

	http.HandleFunc("/tasks", createTaskHandler(store))
	http.HandleFunc("/tasks/", getTaskHandler(store))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "file-archiver: service is up")
	})

	log.Printf("file-archiver is listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func createTaskHandler(store task.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		t := store.Create()
		respondJSON(w, http.StatusCreated, t)
	}
}

func getTaskHandler(store task.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		id := r.URL.Path[len("/tasks/"):]
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		t, err := store.Get(id)
		if err != nil {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		respondJSON(w, http.StatusOK, t)
	}
}

func respondJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

package main

import (
	"encoding/json"
	"file-archiver/internal/task"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

func main() {
	addr := ":" + getenv("PORT", "8080")

	store := task.NewMemoryStore()

	http.HandleFunc("/tasks", createTaskHandler(store))
	http.HandleFunc("/tasks/", taskHandler(store))

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

func taskHandler(store task.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/tasks/")
		parts := strings.Split(rest, "/")

		id := parts[0]
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			if len(parts) != 1 {
				http.NotFound(w, r)
				return
			}
			t, err := store.Get(id)
			if err != nil {
				http.Error(w, "task not found", http.StatusNotFound)
				return
			}
			respondJSON(w, http.StatusOK, t)
		case http.MethodPost:
			if len(parts) != 2 || parts[1] == "items" {
				http.NotFound(w, r)
				return
			}
			addItem(w, r, store, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func addItem(w http.ResponseWriter, r *http.Request, store task.Store, id string) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := validateURL(req.URL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.AddItem(id, req.URL); err != nil {
		switch err {
		case task.ErrNotFound:
			http.Error(w, "task not found", http.StatusNotFound)
		case task.ErrTooManyUtems, task.ErrTaskFinalized:
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid url")
	}
	ext := strings.ToLower(path.Ext(u.Path))
	switch ext {
	case ".pdf", ".jpeg", ".jpg":
		return nil
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
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

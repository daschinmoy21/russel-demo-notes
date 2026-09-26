// notes-api: a tiny in-memory JSON notes service, built to demo Russel deploys.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Note struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

type store struct {
	mu     sync.Mutex
	nextID int
	notes  map[int]Note
}

var (
	db      = &store{nextID: 1, notes: map[int]Note{}}
	started = time.Now()
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	greeting := os.Getenv("GREETING")
	if greeting == "" {
		greeting = "hello from russel"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		host, _ := os.Hostname()
		writeJSON(w, 200, map[string]any{
			"service":  "notes-api",
			"message":  greeting,
			"hostname": host,
			"uptime":   time.Since(started).Round(time.Second).String(),
			"routes":   []string{"GET /health", "GET /notes", "POST /notes", "GET /notes/{id}", "DELETE /notes/{id}"},
		})
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /notes", func(w http.ResponseWriter, r *http.Request) {
		db.mu.Lock()
		out := make([]Note, 0, len(db.notes))
		for _, n := range db.notes {
			out = append(out, n)
		}
		db.mu.Unlock()
		sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
		writeJSON(w, 200, out)
	})

	mux.HandleFunc("POST /notes", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil || body.Text == "" {
			writeJSON(w, 400, map[string]string{"error": `expected {"text": "..."}`})
			return
		}
		db.mu.Lock()
		n := Note{ID: db.nextID, Text: body.Text, CreatedAt: time.Now().UTC()}
		db.notes[n.ID] = n
		db.nextID++
		db.mu.Unlock()
		writeJSON(w, 201, n)
	})

	mux.HandleFunc("GET /notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		db.mu.Lock()
		n, ok := db.notes[id]
		db.mu.Unlock()
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, 200, n)
	})

	mux.HandleFunc("DELETE /notes/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		db.mu.Lock()
		_, ok := db.notes[id]
		delete(db.notes, id)
		db.mu.Unlock()
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		w.WriteHeader(204)
	})

	log.Printf("notes-api listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

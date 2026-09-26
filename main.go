// notes-api: a tiny in-memory JSON notes service, built to demo Russel deploys.
package main

import (
	_ "embed"
	"encoding/json"
	"errors"
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
	path   string // JSON file to persist to; empty means memory only
}

type snapshot struct {
	NextID int    `json:"next_id"`
	Notes  []Note `json:"notes"`
}

// load reads the persisted notes, if any. A missing file is not an error.
func (s *store) load() error {
	if s.path == "" {
		return nil
	}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	var snap snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return err
	}
	for _, n := range snap.Notes {
		s.notes[n.ID] = n
	}
	s.nextID = max(snap.NextID, 1)
	return nil
}

// save writes all notes atomically. Caller holds s.mu.
func (s *store) save() error {
	if s.path == "" {
		return nil
	}
	snap := snapshot{NextID: s.nextID, Notes: make([]Note, 0, len(s.notes))}
	for _, n := range s.notes {
		snap.Notes = append(snap.Notes, n)
	}
	b, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

//go:embed static/index.html
var indexHTML []byte

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
	db.path = os.Getenv("NOTES_FILE")
	if err := db.load(); err != nil {
		log.Fatalf("load %s: %v", db.path, err)
	}
	storage := "memory"
	if db.path != "" {
		storage = db.path
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	mux.HandleFunc("GET /api/info", func(w http.ResponseWriter, r *http.Request) {
		host, _ := os.Hostname()
		db.mu.Lock()
		count := len(db.notes)
		db.mu.Unlock()
		writeJSON(w, 200, map[string]any{
			"service":  "notes-api",
			"message":  greeting,
			"hostname": host,
			"storage":  storage,
			"notes":    count,
			"uptime":   time.Since(started).Round(time.Second).String(),
			"routes":   []string{"GET /", "GET /api/info", "GET /health", "GET /notes", "POST /notes", "GET /notes/{id}", "DELETE /notes/{id}"},
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
		err := db.save()
		db.mu.Unlock()
		if err != nil {
			log.Printf("save: %v", err)
			writeJSON(w, 500, map[string]string{"error": "could not persist note"})
			return
		}
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
		var err error
		if ok {
			err = db.save()
		}
		db.mu.Unlock()
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		if err != nil {
			log.Printf("save: %v", err)
			writeJSON(w, 500, map[string]string{"error": "could not persist delete"})
			return
		}
		w.WriteHeader(204)
	})

	log.Printf("notes-api listening on :%s (storage: %s, %d notes)", port, storage, len(db.notes))
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

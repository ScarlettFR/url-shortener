package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type record struct {
	URL       string    `json:"url"`
	Hits      int64     `json:"hits"`
	CreatedAt time.Time `json:"created_at"`
}

type store struct {
	mu   sync.RWMutex
	data map[string]*record
}

func newStore() *store {
	return &store{data: make(map[string]*record)}
}

func (s *store) put(code, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[code] = &record{URL: url, CreatedAt: time.Now().UTC()}
}

func (s *store) get(code string) (*record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.data[code]
	return r, ok
}

func (s *store) bump(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.data[code]; ok {
		r.Hits++
	}
}

func genCode(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// крайне маловероятно, но молча фоллбэчить на time-based сиду не хочется
		panic(err)
	}
	s := strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
	if len(s) < n {
		return s
	}
	return s[:n]
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type server struct {
	st   *store
	host string
}

func (srv *server) shorten(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL == "" {
		http.Error(w, "bad url", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(body.URL, "http://") && !strings.HasPrefix(body.URL, "https://") {
		http.Error(w, "url must be http(s)", http.StatusBadRequest)
		return
	}

	code := genCode(7)
	srv.st.put(code, body.URL)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":  code,
		"short": srv.host + "/" + code,
	})
}

func (srv *server) redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	rec, ok := srv.st.get(code)
	if !ok {
		http.NotFound(w, r)
		return
	}
	srv.st.bump(code)
	http.Redirect(w, r, rec.URL, http.StatusFound)
}

func (srv *server) stats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	rec, ok := srv.st.get(code)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rec)
}

func main() {
	addr := ":" + envOr("PORT", "8080")
	host := envOr("BASE_URL", "http://localhost"+addr)

	srv := &server{st: newStore(), host: host}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", srv.shorten)
	mux.HandleFunc("GET /stats/{code}", srv.stats)
	mux.HandleFunc("GET /{code}", srv.redirect)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("listening on %s", addr)

	s := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

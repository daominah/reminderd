package httpsvr

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/daominah/reminderd/pkg/logic"
)

var vnTimezone = time.FixedZone("ICT", 7*60*60)

// Server serves the web UI and API endpoints.
type Server struct {
	ConfigStore   logic.ConfigStore
	HistoryReader logic.HistoryReader
	Notifier      logic.Notifier
	Tracker       *logic.UserInputTracker
	FrontendFS    fs.FS
	Port          int
}

func NewServer(configStore logic.ConfigStore, historyReader logic.HistoryReader, frontendFS fs.FS, port int) *Server {
	return &Server{
		ConfigStore:   configStore,
		HistoryReader: historyReader,
		FrontendFS:    frontendFS,
		Port:          port,
	}
}

// Handler returns an http.Handler with all routes registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(s.FrontendFS)))
	mux.HandleFunc("GET /api/history", s.handleGetHistory)
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("POST /api/config", s.handlePostConfig)
	mux.HandleFunc("POST /api/test-notification", s.handleTestNotification)
	return mux
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	addr := fmt.Sprintf(":%d", s.Port)
	log.Printf("web UI listening on http://localhost%s", addr)
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	now := time.Now().In(vnTimezone)
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, vnTimezone)

	start := startOfDay
	var end *time.Time

	if v := r.URL.Query().Get("start"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			http.Error(w, "invalid start time", http.StatusBadRequest)
			return
		}
		start = t
	}
	if v := r.URL.Query().Get("end"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			http.Error(w, "invalid end time", http.StatusBadRequest)
			return
		}
		end = &t
	}

	entries, err := s.HistoryReader.ReadRange(start, end)
	if err != nil {
		http.Error(w, fmt.Sprintf("error reading history: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if entries == nil {
		w.Write([]byte("[]\n"))
		return
	}
	json.NewEncoder(w).Encode(entries)
}

// configResponse is the JSON representation sent to and received from the frontend.
type configResponse struct {
	ContinuousActiveLimit          string `json:"ContinuousActiveLimit"`
	IdleDurationToConsiderBreak    string `json:"IdleDurationToConsiderBreak"`
	KeyboardMouseInputPollInterval string `json:"KeyboardMouseInputPollInterval"`
	NotificationInitialBackoff     string `json:"NotificationInitialBackoff"`
	WebUIPort                      int    `json:"WebUIPort"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.ConfigStore.Load()
	if err != nil {
		http.Error(w, fmt.Sprintf("error loading config: %v", err), http.StatusInternalServerError)
		return
	}
	resp := configResponse{
		ContinuousActiveLimit:          cfg.ContinuousActiveLimit.String(),
		IdleDurationToConsiderBreak:    cfg.IdleDurationToConsiderBreak.String(),
		KeyboardMouseInputPollInterval: cfg.KeyboardMouseInputPollInterval.String(),
		NotificationInitialBackoff:     cfg.NotificationInitialBackoff.String(),
		WebUIPort:                      cfg.WebUIPort,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handlePostConfig(w http.ResponseWriter, r *http.Request) {
	var req configResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	cfg, err := s.ConfigStore.Load()
	if err != nil {
		http.Error(w, fmt.Sprintf("error loading config: %v", err), http.StatusInternalServerError)
		return
	}

	if req.ContinuousActiveLimit != "" {
		if d, err := time.ParseDuration(req.ContinuousActiveLimit); err == nil {
			cfg.ContinuousActiveLimit = d
		}
	}
	if req.IdleDurationToConsiderBreak != "" {
		if d, err := time.ParseDuration(req.IdleDurationToConsiderBreak); err == nil {
			cfg.IdleDurationToConsiderBreak = d
		}
	}
	if req.KeyboardMouseInputPollInterval != "" {
		if d, err := time.ParseDuration(req.KeyboardMouseInputPollInterval); err == nil {
			cfg.KeyboardMouseInputPollInterval = d
		}
	}
	if req.NotificationInitialBackoff != "" {
		if d, err := time.ParseDuration(req.NotificationInitialBackoff); err == nil {
			cfg.NotificationInitialBackoff = d
		}
	}
	if req.WebUIPort > 0 {
		cfg.WebUIPort = req.WebUIPort
	}

	if err := s.ConfigStore.Save(cfg); err != nil {
		http.Error(w, fmt.Sprintf("error saving config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleTestNotification(w http.ResponseWriter, r *http.Request) {
	if s.Notifier == nil {
		http.Error(w, "notifier not available", http.StatusServiceUnavailable)
		return
	}

	var activeDuration time.Duration
	if s.Tracker != nil {
		activeDuration = s.Tracker.ActiveDuration()
	}

	title := "Sat Too Long, Take a Break"
	msg := fmt.Sprintf("You have been active for %s. Take a break!",
		formatActiveDuration(activeDuration))
	s.Notifier.Notify(title, msg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"activeDuration": formatActiveDuration(activeDuration),
	})
}

func formatActiveDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

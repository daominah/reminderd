package httpsvr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/daominah/reminderd/pkg/base"
	"github.com/daominah/reminderd/pkg/logic"
	"github.com/daominah/reminderd/pkg/model"
)

var testFS = fstest.MapFS{
	"index.html": {Data: []byte("<html><body>test</body></html>")},
}

func TestGetIndex_ReturnsHTML(t *testing.T) {
	// GIVEN an HTTP server
	srv := NewServer(
		&logic.MockConfigStore{Cfg: model.DefaultConfig()},
		&logic.MockHistoryReader{},
		testFS,
		20902,
	)

	// WHEN GET / is called
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	// THEN it returns HTML
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected Content-Type text/html, got %q", ct)
	}
}

func TestGetAPIHistory_ReturnsJSON(t *testing.T) {
	// GIVEN a server with history entries
	entries := []model.HistoryEntry{
		{Time: model.FormatTime(time.Date(2026, 4, 2, 10, 0, 0, 0, base.VietnamTimezone)), State: model.Active},
		{Time: model.FormatTime(time.Date(2026, 4, 2, 10, 0, 10, 0, base.VietnamTimezone)), State: model.Idle},
	}
	srv := NewServer(
		&logic.MockConfigStore{Cfg: model.DefaultConfig()},
		&logic.MockHistoryReader{Entries: entries},
		testFS,
		20902,
	)

	// WHEN GET /api/history is called
	req := httptest.NewRequest(http.MethodGet, "/api/history", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	// THEN it returns JSON with 2 entries
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var result []model.HistoryEntry
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
}

func TestGetAPIHistory_AcceptsTimeRange(t *testing.T) {
	// GIVEN a server with history entries
	entries := []model.HistoryEntry{
		{Time: model.FormatTime(time.Date(2026, 4, 2, 14, 0, 0, 0, base.VietnamTimezone)), State: model.Active},
	}
	srv := NewServer(
		&logic.MockConfigStore{Cfg: model.DefaultConfig()},
		&logic.MockHistoryReader{Entries: entries},
		testFS,
		20902,
	)

	// WHEN GET /api/history with start and end params is called
	req := httptest.NewRequest(http.MethodGet,
		"/api/history?start=2026-04-02T12:00:00%2B07:00&end=2026-04-02T16:00:00%2B07:00", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	// THEN it returns 200
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestGetAPIConfig_ReturnsCurrentConfig(t *testing.T) {
	// GIVEN a server with a config store
	cfg := model.DefaultConfig()
	cfg.WebUIPort = 9999
	srv := NewServer(
		&logic.MockConfigStore{Cfg: cfg},
		&logic.MockHistoryReader{},
		testFS,
		9999,
	)

	// WHEN GET /api/config is called
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	// THEN it returns the config as JSON with the correct port
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var result map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	port, ok := result["WebUIPort"]
	if !ok {
		t.Fatal("expected WebUIPort in response")
	}
	if int(port.(float64)) != 9999 {
		t.Errorf("expected WebUIPort 9999, got %v", port)
	}
}

func TestPostAPIConfig_UpdatesConfig(t *testing.T) {
	// GIVEN a server with a config store
	configStore := &logic.MockConfigStore{Cfg: model.DefaultConfig()}
	srv := NewServer(configStore, &logic.MockHistoryReader{}, testFS, 20902)

	// WHEN POST /api/config is called with new values
	body := `{"ContinuousActiveLimit":"30m","WebUIPort":8080}`
	req := httptest.NewRequest(http.MethodPost, "/api/config",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	// THEN the config is updated
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if configStore.Cfg.WebUIPort != 8080 {
		t.Errorf("expected WebUIPort 8080, got %d", configStore.Cfg.WebUIPort)
	}
}

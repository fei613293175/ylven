package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/fei613293175/ylven/backend/internal/platform/health"
)

func TestProcessHealthContract(t *testing.T) {
	services := []struct{name, path string}{
		{"core-api", "/api/health"}, {"ai-runtime", "/runtime/health"},
		{"developer-gateway", "/developer/health"}, {"worker", "/worker/health"},
		{"scheduler", "/scheduler/health"},
	}
	for _, tc := range services {
		h := health.Handler(health.Service{Name: tc.name, Version: "1.0.0", Dependencies: map[string]bool{"config": true}}, tc.path)
		w := httptest.NewRecorder(); h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if w.Code != http.StatusOK { t.Fatalf("%s: status %d", tc.name, w.Code) }
		var body struct { Service string `json:"service"`; Status string `json:"status"`; Version string `json:"version"` }
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil { t.Fatal(err) }
		if body.Service != tc.name || body.Status != "healthy" || body.Version != "1.0.0" { t.Fatalf("%s: %#v", tc.name, body) }
	}
}

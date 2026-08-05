package runtime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthReadinessVersionAndErrorEnvelope(t *testing.T) {
	h := Handler(Service{Name:"core-api", Version:"1.0.0", Dependencies:map[string]bool{"database":false}}, "/api/health")
	for _, path := range []string{"/healthz", "/api/health", "/version"} {
		r := httptest.NewRequest(http.MethodGet, path, nil); w := httptest.NewRecorder(); h.ServeHTTP(w, r)
		if w.Code != http.StatusOK { t.Fatalf("%s: %d", path, w.Code) }
	}
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil); w := httptest.NewRecorder(); h.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable { t.Fatalf("ready status: %d", w.Code) }
	var body map[string]any; if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil { t.Fatal(err) }
	if body["status"] != "unready" { t.Fatalf("body: %v", body) }
}


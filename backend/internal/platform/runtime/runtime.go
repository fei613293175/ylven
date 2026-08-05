package runtime

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

// Service describes the public identity and dependency state of one process.
type Service struct {
	Name        string
	Version     string
	Dependencies map[string]bool
}

type dependencyState struct {
	Name   string `json:"name"`
	Healthy bool   `json:"healthy"`
}

type healthResponse struct {
	Service string             `json:"service"`
	Status  string             `json:"status"`
	Version string             `json:"version"`
	Time    time.Time          `json:"time"`
	Checks  []dependencyState  `json:"checks,omitempty"`
}

type errorBody struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	RequestID string         `json:"request_id"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

type errorEnvelope struct { Error errorBody `json:"error"` }

// Handler exposes the common process endpoints and a feature-specific health path.
func Handler(service Service, featurePath string) http.Handler {
	if service.Version == "" { service.Version = "dev" }
	if service.Dependencies == nil { service.Dependencies = map[string]bool{} }
	mux := http.NewServeMux()
	health := func(w http.ResponseWriter, r *http.Request) {
		writeHealth(w, service, r.URL.Path == "/readyz")
	}
	mux.HandleFunc("/healthz", health)
	mux.HandleFunc("/readyz", health)
	mux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"service": service.Name, "version": service.Version})
	})
	mux.HandleFunc("/internal/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ylven_process_up 1\n"))
	})
	mux.HandleFunc(featurePath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet { writeError(w, r, http.StatusBadRequest, "INVALID_REQUEST", "method not allowed", false); return }
		writeHealth(w, service, false)
	})
	return mux
}

func writeHealth(w http.ResponseWriter, service Service, readiness bool) {
	ok := true
	names := make([]string, 0, len(service.Dependencies))
	for name := range service.Dependencies { names = append(names, name) }
	sort.Strings(names)
	checks := make([]dependencyState, 0, len(names))
	for _, name := range names {
		healthy := service.Dependencies[name]
		checks = append(checks, dependencyState{Name: name, Healthy: healthy})
		if !healthy { ok = false }
	}
	status := "healthy"
	if !ok { status = "degraded" }
	if readiness && !ok { status = "unready" }
	code := http.StatusOK
	if readiness && !ok { code = http.StatusServiceUnavailable }
	writeJSON(w, code, healthResponse{Service: service.Name, Status: status, Version: service.Version, Time: time.Now().UTC(), Checks: checks})
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, retryable bool) {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" { requestID = "local-request" }
	writeJSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: message, RequestID: requestID, Retryable: retryable}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/fei613293175/ylven/backend/internal/identity"
	"github.com/fei613293175/ylven/backend/internal/platform/runtime"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" { addr = ":8080" }
	store, err := identity.NewStore(os.Getenv("IDENTITY_STORE_PATH"))
	if err != nil { log.Fatal(err) }
	mux := http.NewServeMux()
	mux.Handle("/healthz", runtime.Handler(runtime.Service{Name:"core-api", Version:os.Getenv("APP_VERSION"), Dependencies:map[string]bool{"config":true}}, "/api/health"))
	mux.Handle("/readyz", runtime.Handler(runtime.Service{Name:"core-api", Version:os.Getenv("APP_VERSION"), Dependencies:map[string]bool{"config":true}}, "/api/health"))
	mux.Handle("/version", runtime.Handler(runtime.Service{Name:"core-api", Version:os.Getenv("APP_VERSION"), Dependencies:map[string]bool{"config":true}}, "/api/health"))
	mux.Handle("/internal/metrics", runtime.Handler(runtime.Service{Name:"core-api", Version:os.Getenv("APP_VERSION"), Dependencies:map[string]bool{"config":true}}, "/api/health"))
	mux.Handle("/api/health", runtime.Handler(runtime.Service{Name:"core-api", Version:os.Getenv("APP_VERSION"), Dependencies:map[string]bool{"config":true}}, "/api/health"))
	mux.Handle("/", identity.NewAPI(store).Handler())
	log.Fatal(http.ListenAndServe(addr, mux))
}

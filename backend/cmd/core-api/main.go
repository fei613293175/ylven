package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/fei613293175/ylven/backend/internal/identity"
	"github.com/fei613293175/ylven/backend/internal/platform/runtime"
)

func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	store, err := identity.NewStore(os.Getenv("IDENTITY_STORE_PATH"))
	if err != nil {
		log.Fatal(err)
	}
	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelDatabase()
	if err := store.EnablePostgresFromEnvironment(databaseContext); err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	api := identity.NewAPI(store)
	service := runtime.Service{
		Name:         "core-api",
		Version:      os.Getenv("APP_VERSION"),
		Dependencies: map[string]bool{"config": api.ConfigurationOK()},
	}
	health := runtime.Handler(service, "/api/health")
	mux := http.NewServeMux()
	for _, path := range []string{"/healthz", "/readyz", "/health", "/health/ready", "/version", "/internal/metrics", "/api/health"} {
		mux.Handle(path, health)
	}
	mux.Handle("/", api.Handler())
	log.Fatal(http.ListenAndServe(addr, mux))
}

package main

import ("log"; "net/http"; "os"; "github.com/fei613293175/ylven/backend/internal/platform/runtime")
func main() { addr:=os.Getenv("HTTP_ADDR"); if addr=="" { addr=":8084" }; log.Fatal(http.ListenAndServe(addr, runtime.Handler(runtime.Service{Name:"scheduler", Version:os.Getenv("APP_VERSION"), Dependencies:map[string]bool{"lock-store":true}}, "/scheduler/health"))) }

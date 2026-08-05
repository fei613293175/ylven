package telemetry

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

type Logger struct { mu sync.Mutex; sink func([]byte) }
func NewLogger(sink func([]byte)) *Logger { return &Logger{sink:sink} }
func (l *Logger) Event(level, message, requestID string, fields map[string]string) {
	values:=map[string]string{"level":level,"message":message,"request_id":requestID}
	for key,value:=range fields { lowered:=strings.ToLower(key); if strings.Contains(lowered,"secret")||strings.Contains(lowered,"token")||strings.Contains(lowered,"password")||strings.Contains(lowered,"api_key") { value="[REDACTED]" }; values[key]=value }
	data,_:=json.Marshal(values); l.mu.Lock(); defer l.mu.Unlock(); if l.sink!=nil { l.sink(data) }
}

type Metrics struct { requests atomic.Uint64; failures atomic.Uint64 }
func (m *Metrics) ObserveRequest(failed bool) { m.requests.Add(1); if failed {m.failures.Add(1)} }
func (m *Metrics) Handler(w http.ResponseWriter,_ *http.Request){w.Header().Set("Content-Type","text/plain; version=0.0.4");w.WriteHeader(http.StatusOK);_,_=w.Write([]byte("ylven_requests_total "+itoa(m.requests.Load())+"\nylven_request_failures_total "+itoa(m.failures.Load())+"\n"))}
func itoa(value uint64) string { if value==0{return "0"}; var buf [20]byte; i:=len(buf); for value>0 {i--;buf[i]=byte('0'+value%10);value/=10};return string(buf[i:]) }

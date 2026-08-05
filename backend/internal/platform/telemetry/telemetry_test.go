package telemetry

import ("encoding/json"; "net/http/httptest"; "strings"; "testing")
func TestLoggerRedactsSecretsAndMetrics(t *testing.T){var line []byte;l:=NewLogger(func(data []byte){line=data});l.Event("info","probe","r1",map[string]string{"api_key":"secret","user":"ok"});var body map[string]string;if err:=json.Unmarshal(line,&body);err!=nil{t.Fatal(err)};if body["api_key"]!="[REDACTED]"{t.Fatalf("%s",line)};m:=&Metrics{};m.ObserveRequest(false);m.ObserveRequest(true);w:=httptest.NewRecorder();m.Handler(w,httptest.NewRequest("GET","/internal/metrics",nil));if !strings.Contains(w.Body.String(),"ylven_requests_total 2")||!strings.Contains(w.Body.String(),"ylven_request_failures_total 1"){t.Fatal(w.Body.String())}}

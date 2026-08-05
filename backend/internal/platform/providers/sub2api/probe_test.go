package sub2api

import ("context"; "net/http"; "net/http/httptest"; "testing")
func TestProbeNeverClaimsUnconfiguredSuccess(t *testing.T){result:=Client{}.Probe(context.Background());if result.Status!=Unconfigured||result.ErrorCode!="SUB2API_NOT_CONFIGURED"{t.Fatalf("%#v",result)}}
func TestProbeReportsTextAndStream(t *testing.T){calls:=0;c:=Client{Config:Config{Endpoint:"https://example.invalid",APIKey:"configured-in-secret-store"},Request:func(context.Context,bool)error{calls++;return nil}};result:=c.Probe(context.Background());if result.Status!=Succeeded||!result.Text||!result.Streaming||calls!=2{t.Fatalf("%#v calls=%d",result,calls)}}
func TestProbeEndpointMakesUnconfiguredStateExplicit(t *testing.T){w:=httptest.NewRecorder();Client{}.Handler().ServeHTTP(w,httptest.NewRequest(http.MethodPost,"/admin/v1/probes/sub2api/chat",nil));if w.Code!=http.StatusServiceUnavailable{t.Fatalf("status %d",w.Code)}}

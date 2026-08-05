package sub2api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

type Status string
const ( Unconfigured Status = "UNCONFIGURED"; Succeeded Status = "SUCCEEDED"; Failed Status = "FAILED" )
type Config struct { Endpoint string; APIKey string }
type Result struct { Status Status; Text bool; Streaming bool; ErrorCode string }
var ErrUnconfigured = errors.New("SUB2API_NOT_CONFIGURED")

type Client struct { Config Config; Request func(context.Context, bool) error }
func (c Client) Probe(ctx context.Context) Result { if c.Config.Endpoint==""||c.Config.APIKey=="" {return Result{Status:Unconfigured,ErrorCode:ErrUnconfigured.Error()}}; if c.Request==nil {return Result{Status:Failed,ErrorCode:"SUB2API_PROBE_UNAVAILABLE"}}; if err:=c.Request(ctx,false);err!=nil{return Result{Status:Failed,ErrorCode:"SUB2API_TEXT_FAILED"}}; if err:=c.Request(ctx,true);err!=nil{return Result{Status:Failed,Text:true,ErrorCode:"SUB2API_STREAM_FAILED"}};return Result{Status:Succeeded,Text:true,Streaming:true} }

func (c Client) Handler() http.Handler { mux:=http.NewServeMux(); mux.HandleFunc("/admin/v1/probes/sub2api/chat",func(w http.ResponseWriter,r *http.Request){if r.Method!=http.MethodPost{w.WriteHeader(http.StatusBadRequest);return};result:=c.Probe(r.Context());status:=http.StatusOK;if result.Status==Unconfigured{status=http.StatusServiceUnavailable};w.Header().Set("Content-Type","application/json");w.WriteHeader(status);_=json.NewEncoder(w).Encode(result)});return mux }

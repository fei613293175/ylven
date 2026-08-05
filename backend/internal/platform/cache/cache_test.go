package cache

import ("context"; "testing"; "time")
func TestTTLAndBoundedRateLimit(t *testing.T){m:=NewMemory();now:=time.Unix(1,0);if err:=m.Set(context.Background(),"k",[]byte("v"),time.Second,now);err!=nil{t.Fatal(err)};if _,ok,err:=m.Get(context.Background(),"k",now.Add(2*time.Second));err!=nil||ok{t.Fatalf("expired: %v %v",ok,err)};l:=NewLimiter(1,time.Second);if err:=l.Allow("u",now);err!=nil{t.Fatal(err)};if err:=l.Allow("u",now);err!=ErrRateLimited{t.Fatalf("got %v",err)}}

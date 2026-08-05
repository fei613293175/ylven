package events

import ("context"; "testing")
func TestDuplicateEventIsNotDeliveredTwice(t *testing.T){b:=NewMemoryBus();e:=Event{ID:"1",Subject:"jobs",Data:[]byte("x")};if err:=b.Publish(context.Background(),e);err!=nil{t.Fatal(err)};if err:=b.Publish(context.Background(),e);err!=ErrDuplicate{t.Fatalf("got %v",err)};if got:=b.Consume(context.Background(),"jobs");len(got)!=1{t.Fatalf("got %d",len(got))}}

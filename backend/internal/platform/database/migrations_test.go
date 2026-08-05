package database

import ("testing"; "time")
func TestMigrationChecksumAndImmutableRollback(t *testing.T){m:=NewMigrator(); list:=[]Migration{{Version:1,Name:"runtime",SQL:"create"}}; if _,err:=m.Apply(list,time.Unix(1,0));err!=nil{t.Fatal(err)}; if _,err:=m.Apply([]Migration{{Version:1,Name:"runtime",SQL:"changed"}},time.Unix(2,0));err!=ErrChecksumMismatch{t.Fatalf("got %v",err)}; if err:=m.Rollback(1);err!=ErrDowngrade{t.Fatalf("got %v",err)}}

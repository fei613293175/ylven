package storage

import ("context"; "io"; "strings"; "testing")
func TestLocalFilesystemAndExplicitR2Boundary(t *testing.T){root:=t.TempDir();s:=LocalFS{Root:root};if err:=s.Put(context.Background(),"a/b.txt",strings.NewReader("ok"));err!=nil{t.Fatal(err)};f,err:=s.Get(context.Background(),"a/b.txt");if err!=nil{t.Fatal(err)};defer f.Close();body,_:=io.ReadAll(f);if string(body)!="ok"{t.Fatalf("%s",body)};if err:=(R2{}).Put(context.Background(),"x",strings.NewReader("x"));err!=ErrExternalNotConfigured{t.Fatalf("got %v",err)}}

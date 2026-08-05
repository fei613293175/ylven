package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

var ErrExternalNotConfigured = errors.New("external object storage is not configured")
type ObjectStore interface { Put(context.Context,string,io.Reader) error; Get(context.Context,string)(io.ReadCloser,error) }

type LocalFS struct { Root string }
func (s LocalFS) Put(_ context.Context, key string, source io.Reader) error { target:=filepath.Join(s.Root,filepath.Clean("/"+key)); if err:=os.MkdirAll(filepath.Dir(target),0700);err!=nil{return err}; f,err:=os.OpenFile(target,os.O_CREATE|os.O_TRUNC|os.O_WRONLY,0600);if err!=nil{return err};defer f.Close();_,err=io.Copy(f,source);return err }
func (s LocalFS) Get(_ context.Context,key string)(io.ReadCloser,error){target:=filepath.Join(s.Root,filepath.Clean("/"+key));return os.Open(target)}
type R2 struct { Endpoint string; Bucket string }
func (r R2) Put(context.Context,string,io.Reader) error { return ErrExternalNotConfigured }
func (r R2) Get(context.Context,string)(io.ReadCloser,error){ return nil,ErrExternalNotConfigured }

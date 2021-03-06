package fs

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/bitrainforest/datastore/store"
)

type FS struct {
	path string
}

func New(path string) (store.Store, error) {
	if err := initPath(path); err != nil {
		return nil, err
	}
	return &FS{path: path}, nil
}

func (s *FS) CreateBucket(_ context.Context, bucket string) error {
	bucketPath := path.Join(s.path, bucket)
	return os.Mkdir(bucketPath, 0755)
}

func (s *FS) Read(_ context.Context, bucket, key string) ([]byte, error) {
	file, err := os.Open(path.Join(s.path, bucket, key))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ioutil.ReadAll(file)
}

// ReadStream is read as a stream, which the caller must close when finished.
func (s *FS) ReadStream(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	file, err := os.Open(path.Join(s.path, bucket, key))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (s *FS) Write(_ context.Context, bucket, key string, value []byte) error {
	key = path.Join(s.path, bucket, key)
	if err := s.preparePath(key); err != nil {
		return err
	}
	return ioutil.WriteFile(key, value, 0600)
}

func (s *FS) WriteStream(_ context.Context, bucket, key string, value io.Reader) error {
	key = path.Join(s.path, bucket, key)
	if err := s.preparePath(key); err != nil {
		return err
	}
	file, err := os.Create(key)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, value)
	return err
}

func (s *FS) Delete(_ context.Context, bucket, key string) error {
	return os.Remove(path.Join(s.path, bucket, key))
}

func (s *FS) Copy(_ context.Context, bucket, from, to string) error {
	buf := make([]byte, 10240)
	fromFile, err := os.Open(path.Join(s.path, bucket, from))
	if err != nil {
		return err
	}

	defer fromFile.Close() // nolint: errcheck

	toFile, err := os.Create(path.Join(s.path, bucket, to))
	if err != nil {
		return err
	}

	defer toFile.Close() // nolint: errcheck

	for {
		var n int
		n, err = fromFile.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		if _, err = toFile.Write(buf[:n]); err != nil {
			return err
		}
	}

	return err
}

var _ store.Store = (*FS)(nil)

func initPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(path, 0755); err != nil {
				return err
			}
			s, err = os.Stat(path)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if !s.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}

	return nil
}

func (s *FS) preparePath(key string) error {
	sp := strings.Split(key, "/")
	dir := strings.Join(sp[:len(sp)-1], "/")
	if err := initPath(dir); err != nil {
		return err
	}
	return nil
}

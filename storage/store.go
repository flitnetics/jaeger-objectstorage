package storage

import (
        "io"

        "github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
)

var (
	_ shared.StoragePlugin = (*Store)(nil)
	_ io.Closer            = (*Store)(nil)
)

type Store struct {
	reader *Reader
	writer *Writer
}

func NewStore(logger hclog.Logger) (*Store, func() error, error) {
	reader := NewReader("test", logger)
	writer := NewWriter("test")
	store := &Store{
		reader: reader,
		writer: writer,
	}

	return store, store.Close, nil
}

func (s *Store) Close() error {
	return s.writer.Close()
}

func (s *Store) SpanReader() spanstore.Reader {
	return s.reader
}

func (s *Store) SpanWriter() spanstore.Writer {
	return s.writer
}

func (s *Store) DependencyReader() dependencystore.Reader {
	return s.reader
}	

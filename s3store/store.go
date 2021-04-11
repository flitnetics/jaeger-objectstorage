package s3store

import (
	"io"

	"github.com/go-pg/pg/v9"
	hclog "github.com/hashicorp/go-hclog"
        "jaeger-s3/config"

	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

var (
	_ shared.StoragePlugin = (*Store)(nil)
	_ io.Closer            = (*Store)(nil)
)

type Store struct {
	db     *pg.DB
	reader *Reader
	writer *Writer
}


func NewStore(conf *config.Configuration, logger hclog.Logger) (*Store, func() error, error) {
	db := pg.Connect(&pg.Options{
		Addr:     conf.Host,
		User:     conf.Username,
		Password: conf.Password,
	}) 

	reader := NewReader(db, logger)
	writer := NewWriter(db, logger)

	store := &Store{
		db:     db,
		reader: reader,
		writer: writer,
	}

	return store, store.Close, nil
}

// Close writer and DB
func (s *Store) Close() error {
	err2 := s.writer.Close()
	err1 := s.db.Close()
	//s.reader.Close()
	if err1 != nil {
		return err1
	}
	return err2
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

package s3store

import (
        "io/ioutil"
	"io"
        "log"
        "path"

	"github.com/go-pg/pg/v9"
	hclog "github.com/hashicorp/go-hclog"
        "jaeger-s3/config"
        "jaeger-s3/config/types"

        "github.com/grafana/loki/pkg/util/validation"

        "github.com/cortexproject/cortex/pkg/util/flagext"
        "github.com/cortexproject/cortex/pkg/chunk/storage"
        cortex_local "github.com/cortexproject/cortex/pkg/chunk/local"
        util_log "github.com/cortexproject/cortex/pkg/util/log"

	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
        lstore "github.com/grafana/loki/pkg/storage"
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

func NewStore(conf *config.Configuration, cfg *types.Config, logger hclog.Logger) (*Store, func() error, error) {
	db := pg.Connect(&pg.Options{
		Addr:     conf.Host,
		User:     conf.Username,
		Password: conf.Password,
	}) 

        lstore.RegisterCustomIndexClients(&cfg.StorageConfig, nil)

        tempDir, err := ioutil.TempDir("", "boltdb-shippers")
        if err != nil {
                log.Println("tempDir failure %s", err)
        }

        limits, err := validation.NewOverrides(validation.Limits{}, nil)
        if err != nil {
                log.Println("limits failure %s", err)
        }

        // config for BoltDB Shipper
        boltdbShipperConfig := cfg.StorageConfig.BoltDBShipperConfig
        flagext.DefaultValues(&boltdbShipperConfig)

        kconfig := &lstore.Config{
                Config: storage.Config{
                        AWSStorageConfig: cfg.StorageConfig.AWSStorageConfig,
                        FSConfig: cortex_local.FSConfig{Directory: path.Join(tempDir, "chunks")},
                },
                BoltDBShipperConfig: boltdbShipperConfig,
        }

        chunkStore, err := storage.NewStore(
                kconfig.Config,
                cfg.ChunkStoreConfig,
                cfg.SchemaConfig.SchemaConfig,
                limits,
                nil,
                nil,
                util_log.Logger,
        )

        if err != nil {
                log.Println("chunkStore error: %s", err)
        }

        //log.Println("chunkStore: %s", chunkStore)
        dstore, err := lstore.NewStore(*kconfig, cfg.SchemaConfig, chunkStore, nil)
        if err != nil {
               log.Println("store error: %s", err)
        }

	reader := NewReader(db, cfg, dstore, logger)
	writer := NewWriter(db, cfg, dstore, logger)

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

package s3store

import (
	"io"
        "log"

	hclog "github.com/hashicorp/go-hclog"
        "jaeger-s3/config"
        "jaeger-s3/config/types"

        "github.com/grafana/loki/pkg/util/validation"

        "github.com/cortexproject/cortex/pkg/util/flagext"
        "github.com/cortexproject/cortex/pkg/chunk/storage"
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
	reader *Reader
	writer *Writer
}

func NewStore(conf *config.Configuration, cfg *types.Config, logger hclog.Logger) (*Store, func() error, error) {
        lstore.RegisterCustomIndexClients(&cfg.StorageConfig, nil)

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
                        FSConfig: cfg.StorageConfig.FSConfig,
                        AzureStorageConfig: cfg.StorageConfig.AzureStorageConfig,
                        GCPStorageConfig: cfg.StorageConfig.GCPStorageConfig,
                        GCSConfig: cfg.StorageConfig.GCSConfig, 
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

	reader := NewReader(cfg, dstore, logger)
	writer := NewWriter(cfg, dstore, logger)

	store := &Store{
		reader: reader,
		writer: writer,
	}

	return store, store.Close, nil
}

// Close writer and DB
func (s *Store) Close() error {
	err2 := s.writer.Close()
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

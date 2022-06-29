package s3store

import (
	"io"
        "log"
        "time"
        "errors"
        "fmt"

	hclog "github.com/hashicorp/go-hclog"
        "jaeger-s3/config"
        "jaeger-s3/config/types"

        "github.com/cortexproject/cortex/pkg/util/validation"
        "github.com/cortexproject/cortex/pkg/util/flagext"
        "github.com/cortexproject/cortex/pkg/chunk/storage"
        util_log "github.com/cortexproject/cortex/pkg/util/log"

	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
        lstore "github.com/grafana/loki/pkg/storage"
        "github.com/cortexproject/cortex/pkg/chunk/cache"
        "github.com/cortexproject/cortex/pkg/chunk"
        "github.com/grafana/loki/pkg/storage/stores/shipper/uploads"

        "github.com/grafana/loki/pkg/storage/stores/shipper"
)

var (
	_ shared.StoragePlugin = (*Store)(nil)
	_ io.Closer            = (*Store)(nil)
)

// The various modules that make up Loki.
const (
        Ingester        string = "ingester"
        All             string = "all"
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

        // If RF > 1 and current or upcoming index type is boltdb-shipper then disable index dedupe and write dedupe cache.
        // This is to ensure that index entries are replicated to all the boltdb files in ingesters flushing replicated data.
        if cfg.Ingester.LifecyclerConfig.RingConfig.ReplicationFactor > 1 && lstore.UsingBoltdbShipper(cfg.SchemaConfig.Configs) {
                cfg.ChunkStoreConfig.DisableIndexDeduplication = true
                cfg.ChunkStoreConfig.WriteDedupeCacheConfig = cache.Config{}
        }

        if lstore.UsingBoltdbShipper(cfg.SchemaConfig.Configs) {
                cfg.StorageConfig.BoltDBShipperConfig.IngesterName = cfg.Ingester.LifecyclerConfig.ID
                switch cfg.Target {
                case Ingester:
                        // We do not want ingester to unnecessarily keep downloading files
                        cfg.StorageConfig.BoltDBShipperConfig.Mode = shipper.ModeWriteOnly
                        // Use fifo cache for caching index in memory.
                        cfg.StorageConfig.IndexQueriesCacheConfig = cache.Config{
                                EnableFifoCache: true,
                                Fifocache: cache.FifoCacheConfig{
                                        MaxSizeBytes: "200 MB",
                                        // We snapshot the index in ingesters every minute for reads so reduce the index cache validity by a minute.
                                        // This is usually set in StorageConfig.IndexCacheValidity but since this is exclusively used for caching the index entries,
                                        // I(Sandeep) am setting it here which also helps reduce some CPU cycles and allocations required for
                                        // unmarshalling the cached data to check the expiry.
                                        Validity: cfg.StorageConfig.IndexCacheValidity - 1*time.Minute,
                                },
                        }
                        cfg.StorageConfig.BoltDBShipperConfig.IngesterDBRetainPeriod = boltdbShipperQuerierIndexUpdateDelay(cfg) + 2*time.Minute
                default:
                        cfg.StorageConfig.BoltDBShipperConfig.Mode = shipper.ModeReadWrite
                        cfg.StorageConfig.BoltDBShipperConfig.IngesterDBRetainPeriod = boltdbShipperQuerierIndexUpdateDelay(cfg) + 2*time.Minute
                }
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

        if lstore.UsingBoltdbShipper(cfg.SchemaConfig.Configs) {
                boltdbShipperMinIngesterQueryStoreDuration := boltdbShipperMinIngesterQueryStoreDuration(cfg)
                switch cfg.Target {
                case All:
                        // We want ingester to also query the store when using boltdb-shipper but only when running with target All.
                        // We do not want to use AsyncStore otherwise it would start spiraling around doing queries over and over again to the ingesters and store.
                        // ToDo: See if we can avoid doing this when not running loki in clustered mode.
                        cfg.Ingester.QueryStore = true
                        boltdbShipperConfigIdx := lstore.ActivePeriodConfig(cfg.SchemaConfig.Configs)
                        if cfg.SchemaConfig.Configs[boltdbShipperConfigIdx].IndexType != shipper.BoltDBShipperType {
                                boltdbShipperConfigIdx++
                        }
                        mlb, err := calculateMaxLookBack(cfg.SchemaConfig.Configs[boltdbShipperConfigIdx], cfg.Ingester.QueryStoreMaxLookBackPeriod,
                                boltdbShipperMinIngesterQueryStoreDuration)
                        if err != nil {
                                return store, store.Close, nil
                        }
                        cfg.Ingester.QueryStoreMaxLookBackPeriod = mlb
                }
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

// boltdbShipperQuerierIndexUpdateDelay returns duration it could take for queriers to serve the index since it was uploaded.
// It also considers index cache validity because a querier could have cached index just before it was going to resync which means
// it would keep serving index until the cache entries expire.
func boltdbShipperQuerierIndexUpdateDelay(cfg *types.Config) time.Duration {
        return cfg.StorageConfig.IndexCacheValidity + cfg.StorageConfig.BoltDBShipperConfig.ResyncInterval
}

func calculateMaxLookBack(pc chunk.PeriodConfig, maxLookBackConfig, minDuration time.Duration) (time.Duration, error) {
        if pc.ObjectType != shipper.FilesystemObjectStoreType && maxLookBackConfig.Nanoseconds() != 0 {
                return 0, errors.New("it is an error to specify a non zero `query_store_max_look_back_period` value when using any object store other than `filesystem`")
        }

        if maxLookBackConfig == 0 {
                // If the QueryStoreMaxLookBackPeriod is still it's default value of 0, set it to the minDuration.
                return minDuration, nil
        } else if maxLookBackConfig > 0 && maxLookBackConfig < minDuration {
                // If the QueryStoreMaxLookBackPeriod is > 0 (-1 is allowed for infinite), make sure it's at least greater than minDuration or throw an error
                return 0, fmt.Errorf("the configured query_store_max_look_back_period of '%v' is less than the calculated default of '%v' "+
                        "which is calculated based on the max_chunk_age + 15 minute boltdb-shipper interval + 15 min additional buffer.  Increase this value"+
                        "greater than the default or remove it from the configuration to use the default", maxLookBackConfig, minDuration)
        }
        return maxLookBackConfig, nil
} 

// boltdbShipperIngesterIndexUploadDelay returns duration it could take for an index file containing id of a chunk to be uploaded to the shared store since it got flushed.
func boltdbShipperIngesterIndexUploadDelay() time.Duration {
        return uploads.ShardDBsByDuration + shipper.UploadInterval
}

// boltdbShipperMinIngesterQueryStoreDuration returns minimum duration(with some buffer) ingesters should query their stores to
// avoid missing any logs or chunk ids due to async nature of BoltDB Shipper.
func boltdbShipperMinIngesterQueryStoreDuration(cfg *types.Config) time.Duration {
        return cfg.Ingester.MaxChunkAge + boltdbShipperIngesterIndexUploadDelay() + boltdbShipperQuerierIndexUpdateDelay(cfg) + 2*time.Minute
}

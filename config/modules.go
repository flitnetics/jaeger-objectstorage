package config

import (
	"errors"
	"fmt"
	"time"
        "os"

	"github.com/cortexproject/cortex/pkg/chunk"
        "github.com/cortexproject/cortex/pkg/chunk/storage"
	"github.com/cortexproject/cortex/pkg/util/services"
        util_log "github.com/cortexproject/cortex/pkg/util/log"

	"github.com/prometheus/client_golang/prometheus"

        "github.com/grafana/loki/pkg/storage/stores/shipper/compactor"
	"github.com/grafana/loki/pkg/storage/stores/shipper"
	"github.com/grafana/loki/pkg/storage/stores/shipper/uploads"

        "github.com/go-kit/kit/log/level"
)

const maxChunkAgeForTableManager = 12 * time.Hour

// The various modules that make up Loki.
const (
	Compactor       string = "compactor"
        TableManager    string = "table-manager"
        All             string = "all"
)

// Placeholder limits type to pass to cortex frontend
type disabledShuffleShardingLimits struct{}

func (disabledShuffleShardingLimits) MaxQueriersPerUser(userID string) int { return 0 }

func (t *Loki) initTableManager() (services.Service, error) {
        err := t.cfg.SchemaConfig.Load()
        if err != nil {
                return nil, err
        }

        // Assume the newest config is the one to use
        lastConfig := &t.cfg.SchemaConfig.Configs[len(t.cfg.SchemaConfig.Configs)-1]

        if (t.cfg.TableManager.ChunkTables.WriteScale.Enabled ||
                t.cfg.TableManager.IndexTables.WriteScale.Enabled ||
                t.cfg.TableManager.ChunkTables.InactiveWriteScale.Enabled ||
                t.cfg.TableManager.IndexTables.InactiveWriteScale.Enabled ||
                t.cfg.TableManager.ChunkTables.ReadScale.Enabled ||
                t.cfg.TableManager.IndexTables.ReadScale.Enabled ||
                t.cfg.TableManager.ChunkTables.InactiveReadScale.Enabled ||
                t.cfg.TableManager.IndexTables.InactiveReadScale.Enabled) &&
                t.cfg.StorageConfig.AWSStorageConfig.Metrics.URL == "" {
                level.Error(util_log.Logger).Log("msg", "WriteScale is enabled but no Metrics URL has been provided")
                os.Exit(1)
        }

        reg := prometheus.WrapRegistererWith(prometheus.Labels{"component": "table-manager-store"}, prometheus.DefaultRegisterer)

        tableClient, err := storage.NewTableClient(lastConfig.IndexType, t.cfg.StorageConfig.Config, reg)
        if err != nil {
                return nil, err
        }

        bucketClient, err := storage.NewBucketClient(t.cfg.StorageConfig.Config)
        util_log.CheckFatal("initializing bucket client", err)

        t.tableManager, err = chunk.NewTableManager(t.cfg.TableManager, t.cfg.SchemaConfig.SchemaConfig, maxChunkAgeForTableManager, tableClient, bucketClient, nil, prometheus.DefaultRegisterer)
        if err != nil {
                return nil, err
        }

        return t.tableManager, nil
}

func (t *Loki) initCompactor() (services.Service, error) {
	var err error
	t.compactor, err = compactor.NewCompactor(t.cfg.CompactorConfig, t.cfg.StorageConfig.Config, prometheus.DefaultRegisterer)
	if err != nil {
		return nil, err
	}

	return t.compactor, nil
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

func calculateAsyncStoreQueryIngestersWithin(queryIngestersWithinConfig, minDuration time.Duration) time.Duration {
	// 0 means do not limit queries, we would also not limit ingester queries from AsyncStore.
	if queryIngestersWithinConfig == 0 {
		return 0
	}

	if queryIngestersWithinConfig < minDuration {
		return minDuration
	}
	return queryIngestersWithinConfig
}

// boltdbShipperQuerierIndexUpdateDelay returns duration it could take for queriers to serve the index since it was uploaded.
// It also considers index cache validity because a querier could have cached index just before it was going to resync which means
// it would keep serving index until the cache entries expire.
func boltdbShipperQuerierIndexUpdateDelay(cfg Config) time.Duration {
	return cfg.StorageConfig.IndexCacheValidity + cfg.StorageConfig.BoltDBShipperConfig.ResyncInterval
}

// boltdbShipperIngesterIndexUploadDelay returns duration it could take for an index file containing id of a chunk to be uploaded to the shared store since it got flushed.
func boltdbShipperIngesterIndexUploadDelay() time.Duration {
	return uploads.ShardDBsByDuration + shipper.UploadInterval
}

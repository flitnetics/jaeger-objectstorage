package config

import (
        lstore "github.com/grafana/loki/pkg/storage"
        "github.com/grafana/loki/pkg/storage/stores/shipper/compactor"
 
        "github.com/pkg/errors"

	"github.com/spf13/viper"
        "flag"
        "log"

        "github.com/cortexproject/cortex/pkg/chunk/aws"
        "github.com/cortexproject/cortex/pkg/chunk/azure"
        "github.com/cortexproject/cortex/pkg/chunk/gcp"
        "github.com/cortexproject/cortex/pkg/chunk/local"
	"github.com/cortexproject/cortex/pkg/chunk"
        "github.com/cortexproject/cortex/pkg/util/modules"
        "github.com/cortexproject/cortex/pkg/util/services"
        "github.com/cortexproject/cortex/pkg/ring/kv/memberlist"
        "github.com/cortexproject/cortex/pkg/util/runtimeconfig"

        "github.com/grafana/loki/pkg/ingester/client"
        "github.com/grafana/loki/pkg/util/validation"
        "github.com/grafana/loki/pkg/util/runtime"

        "jaeger-s3/s3store/ingester"
)

// Configuration describes the options to customize the storage behavior
type Configuration struct {

	AWSStorageConfig       aws.StorageConfig      `yaml:"aws"`
        AzureStorageConfig     azure.BlobStorageConfig `yaml:"azure"`
        GCPStorageConfig       gcp.Config              `yaml:"bigtable"`
        GCSConfig              gcp.GCSConfig           `yaml:"gcs"`
        FSConfig               local.FSConfig          `yaml:"filesystem"`
	StorageConfig          string                  `yaml:"storage_config"`
}

type Config struct {
	Target      string `yaml:"target,omitempty"`
	AuthEnabled bool   `yaml:"auth_enabled,omitempty"`
	HTTPPrefix  string `yaml:"http_prefix"`

	StorageConfig    lstore.Config               `yaml:"storage_config,omitempty"`
	ChunkStoreConfig chunk.StoreConfig           `yaml:"chunk_store_config,omitempty"`
	TableManager     chunk.TableManagerConfig    `yaml:"table_manager,omitempty"`
	SchemaConfig     lstore.SchemaConfig         `yaml:"schema_config,omitempty"`
	LimitsConfig     validation.Limits           `yaml:"limits_config,omitempty"`
	CompactorConfig  compactor.Config            `yaml:"compactor,omitempty"`
        Ingester         ingester.Config             `yaml:"ingester,omitempty"`
        IngesterClient   client.Config               `yaml:"ingester_client,omitempty"`
}

// Loki is the root datastructure for Loki.
type Loki struct {
        cfg *Config

        // set during initialization
        ModuleManager *modules.Manager
        serviceMap    map[string]services.Service

        compactor       *compactor.Compactor
        tableManager    *chunk.TableManager
        ingester        *ingester.Ingester
        overrides       *validation.Overrides
        tenantConfigs   *runtime.TenantConfigs
        store           lstore.Store
        runtimeConfig   *runtimeconfig.Manager
        memberlistKV    *memberlist.KVInitService
}

func (c *Config) Validate() error {
        if err := c.SchemaConfig.Validate(); err != nil {
                log.Println("schema: %s", c.SchemaConfig)
                log.Println("invalid schema config")
        }
        if err := c.StorageConfig.Validate(); err != nil {
                log.Println("invalid storage config")
        }
        if err := c.TableManager.Validate(); err != nil {
                return errors.Wrap(err, "invalid tablemanager config")
        }
        return nil
}

func (c *Config) RegisterFlags(f *flag.FlagSet) {
	f.BoolVar(&c.AuthEnabled, "auth.enabled", true, "Set to false to disable auth.")

	c.StorageConfig.RegisterFlags(f)
	c.ChunkStoreConfig.RegisterFlags(f)
	c.SchemaConfig.RegisterFlags(f)
	c.TableManager.RegisterFlags(f)
	c.LimitsConfig.RegisterFlags(f)
	c.CompactorConfig.RegisterFlags(f)
}

func (t *Loki) setupModuleManager() error {
        mm := modules.NewManager()

        mm.RegisterModule(Compactor, t.initCompactor)
        mm.RegisterModule(TableManager, t.initTableManager)
        mm.RegisterModule(Ingester, t.initIngester)

        // Add dependencies
        deps := map[string][]string{
                //Compactor:       {Server}, // not needed to run server port/daemon
        }

        // If we are running Loki with boltdb-shipper as a single binary, without clustered mode(which should always be the case when using inmemory ring),
        // we should start compactor as well for better user experience.
        if lstore.UsingBoltdbShipper(t.cfg.SchemaConfig.Configs) {
                deps[All] = append(deps[All], Compactor)
        }

        for mod, targets := range deps {
                if err := mm.AddDependency(mod, targets...); err != nil {
                        return err
                }
        }

        t.ModuleManager = mm

        return nil
}

// InitFromViper initializes the options struct with values from Viper
func (c *Configuration) InitFromViper(v *viper.Viper) {}

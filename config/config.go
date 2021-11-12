package config

import (
        lstore "github.com/grafana/loki/pkg/storage"
        "github.com/grafana/loki/pkg/storage/stores/shipper/compactor"
 
	"github.com/spf13/viper"
        "flag"
        "log"

        "github.com/cortexproject/cortex/pkg/chunk/aws"
        "github.com/cortexproject/cortex/pkg/chunk/azure"
        "github.com/cortexproject/cortex/pkg/chunk/gcp"
        "github.com/cortexproject/cortex/pkg/chunk/local"
	"github.com/cortexproject/cortex/pkg/chunk"

       "github.com/grafana/loki/pkg/util/validation"
)

// Configuration describes the options to customize the storage behavior
type Configuration struct {

	AWSStorageConfig  aws.StorageConfig      `yaml:"aws"`
        AzureStorageConfig     azure.BlobStorageConfig `yaml:"azure"`
        GCPStorageConfig       gcp.Config              `yaml:"bigtable"`
        GCSConfig              gcp.GCSConfig           `yaml:"gcs"`
        FSConfig               local.FSConfig          `yaml:"filesystem"`
	StorageConfig    string       `yaml:"storage_config"`
}

type Config struct {
	Target      string `yaml:"target,omitempty"`
	AuthEnabled bool   `yaml:"auth_enabled,omitempty"`
	HTTPPrefix  string `yaml:"http_prefix"`

	StorageConfig    lstore.Config              `yaml:"storage_config,omitempty"`
	ChunkStoreConfig chunk.StoreConfig           `yaml:"chunk_store_config,omitempty"`
	TableManager     chunk.TableManagerConfig    `yaml:"table_manager,omitempty"`
	SchemaConfig     lstore.SchemaConfig        `yaml:"schema_config,omitempty"`
	LimitsConfig     validation.Limits           `yaml:"limits_config,omitempty"`
	CompactorConfig  compactor.Config            `yaml:"compactor,omitempty"`
}

func (c *Config) Validate() error {
        if err := c.SchemaConfig.Validate(); err != nil {
                log.Println("schema: %s", c.SchemaConfig)
                log.Println("invalid schema config")
        }
        if err := c.StorageConfig.Validate(); err != nil {
                log.Println("invalid storage config")
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

// InitFromViper initializes the options struct with values from Viper
func (c *Configuration) InitFromViper(v *viper.Viper) {}

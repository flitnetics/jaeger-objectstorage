package config

import (
        lstore "jaeger-s3/storage"
        "jaeger-s3/storage/stores/shipper/compactor"
 
	"github.com/spf13/viper"
        "flag"
        "log"

	"github.com/cortexproject/cortex/pkg/chunk"

	"github.com/grafana/loki/pkg/util/validation"
)

const (
	dbPrefix = "db."

	flagHost     = dbPrefix + "host"
	flagUsername = dbPrefix + "username"
	flagPassword = dbPrefix + "password"
        flagDatabase = dbPrefix + "database"
        flagAwsConfig = dbPrefix + "aws"
        flagStorageConfig = dbPrefix + "storage_config"
)

// Configuration describes the options to customize the storage behavior
type Configuration struct {
	// TCP host:port or Unix socket depending on Network.
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
        Database string `yaml:"database"`

	AWSStorageConfig string       `yaml:"aws"`
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
        if err := c.StorageConfig.BoltDBShipperConfig.Validate(); err != nil {
                log.Println("invalid boltdb-shipper config")
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
func (c *Configuration) InitFromViper(v *viper.Viper) {
	c.Host = v.GetString(flagHost)
	if len(c.Host) == 0 {
		c.Host = "localhost:5432"
	}
	c.Username = v.GetString(flagUsername)
	if len(c.Username) == 0 {
		c.Username = "postgres"
	}
	c.Password = v.GetString(flagPassword)
	if len(c.Password) == 0 {
		c.Password = "changeme"
	}
}
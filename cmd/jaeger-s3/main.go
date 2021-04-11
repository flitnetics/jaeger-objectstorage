package main

import (
	"flag"
	"os"
	"sort"
        "log"

        "github.com/cortexproject/cortex/pkg/util/flagext"
 
	"github.com/hashicorp/go-hclog"
	"jaeger-s3/s3store"
        "jaeger-s3/config"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
        "github.com/spf13/viper"
        "github.com/grafana/loki/pkg/cfg"
)

var configPath string

type Config struct {
	config.Config     `yaml:",inline"`
	printVersion    bool
	verifyConfig    bool
	printConfig     bool
	logConfig       bool
	configFile      string
	configExpandEnv bool
}

func (c *Config) RegisterFlags(f *flag.FlagSet) {
	f.BoolVar(&c.printVersion, "version", false, "Print this builds version information")
	f.BoolVar(&c.verifyConfig, "verify-config", false, "Verify config file and exits")
	f.BoolVar(&c.printConfig, "print-config-stderr", false, "Dump the entire Loki config object to stderr")
	f.BoolVar(&c.logConfig, "log-config-reverse-order", false, "Dump the entire Loki config object at Info log "+
		"level with the order reversed, reversing the order makes viewing the entries easier in Grafana.")
	f.StringVar(&c.configFile, "config.file", "", "yaml file to load")
	f.BoolVar(&c.configExpandEnv, "config.expand-env", false, "Expands ${var} in config according to the values of the environment variables.")
	c.Config.RegisterFlags(f)
}

func (c *Config) Clone() flagext.Registerer {
        return func(c Config) *Config {
                return &c
        }(*c)
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "jaeger-s3",
		Level: hclog.Warn, // Jaeger only captures >= Warn, so don't bother logging below Warn
	})

        v := viper.New()
        v.AutomaticEnv()

        v.SetConfigFile("./config-example.yaml")
        v.ReadInConfig()

        var mconfig Config
        log.Println("config: %s", mconfig)
        if err := cfg.Parse(&mconfig); err != nil {
                log.Println("failed to parse config %s", err)
        }

        log.Println("bootup config: %s", &mconfig)

        conf := config.Configuration{}
        conf.InitFromViper(v)

        environ := os.Environ()
        sort.Strings(environ)
        for _, env := range environ {
                logger.Warn(env)
        }

	var store shared.StoragePlugin
	var closeStore func() error
	var err error

	store, closeStore, err = s3store.NewStore(&conf, logger)

	grpc.Serve(store)

	if err = closeStore(); err != nil {
		logger.Error("failed to close store", "error", err)
		os.Exit(1)
	}
}

package main

import (
	"flag"
	"os"
	"sort"
	"strings"
 
	"github.com/hashicorp/go-hclog"
	"jaeger-s3/s3store"
        "jaeger-s3/config"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
        "github.com/spf13/viper"
)

var configPath string

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "jaeger-s3",
		Level: hclog.Warn, // Jaeger only captures >= Warn, so don't bother logging below Warn
	})

        flag.StringVar(&configPath, "config", "", "The absolute path to the S3 plugin's configuration file")
        flag.Parse()

        v := viper.New()
        v.AutomaticEnv()
        v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

        if configPath != "" {
                v.SetConfigFile(configPath)

                err := v.ReadInConfig()
                if err != nil {
                        logger.Error("failed to parse configuration file", "error", err)
                        os.Exit(1)
                }
        }

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

package main

import (
	"os"
	"sort"
        "log"

	"github.com/hashicorp/go-hclog"
	"jaeger-s3/s3store"
        "jaeger-s3/config"
        "jaeger-s3/config/types"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
        "github.com/spf13/viper"
        "github.com/grafana/loki/pkg/cfg"
)

var configPath string

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "jaeger-s3",
		Level: hclog.Warn, // Jaeger only captures >= Warn, so don't bother logging below Warn
	})

        v := viper.New()
        v.AutomaticEnv()

        v.SetConfigFile("./config-example.yaml")
        v.ReadInConfig()

        var mconfig types.Config
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

	//var store shared.PluginServices
	var closeStore func() error
	var err error

	store, closeStore, err := s3store.NewStore(&conf, &mconfig, logger)

	grpc.Serve(&shared.PluginServices{
                Store: store,
        })

	if err = closeStore(); err != nil {
		logger.Error("failed to close store", "error", err)
		os.Exit(1)
	}
}

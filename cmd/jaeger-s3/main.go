package main

import (
        "os"
 
	"github.com/hashicorp/go-hclog"
	"jaeger-s3/storage"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "jaeger-s3",
		Level: hclog.Warn, // Jaeger only captures >= Warn, so don't bother logging below Warn
	})

	var store shared.StoragePlugin
	var closeStore func() error
	var err error

	store, closeStore, err = storage.NewStore(logger)

	grpc.Serve(store)

	if err = closeStore(); err != nil {
		logger.Error("failed to close store", "error", err)
		os.Exit(1)
	}
}

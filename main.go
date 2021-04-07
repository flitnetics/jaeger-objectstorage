package main

import (
    "flag"
    "github.com/hashicorp/go-plugin"
    "google.golang.org/grpc"
    "github.com/muhammadn/jaeger-s3/storage/s3"

    "github.com/jaegertracing/jaeger/plugin/storage/grpc"
    "github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
    "github.com/jaegertracing/jaeger/storage/dependencystore"
    "github.com/jaegertracing/jaeger/storage/spanstore"
)
    
type s3StorePlugin struct {
   	store  *s3.Store
        archiveStore *s3.Store
}

func main() {
    var configPath string
    flag.StringVar(&configPath, "config", "", "A path to the plugin's configuration file")
    flag.Parse()

    plugin := s3StorePlugin{
        store:        s3.NewStore(),
        archiveStore: s3.NewStore(),
    }
    
    grpc.Serve(&shared.PluginServices{
        Store:        plugin,
	ArchiveStore: plugin,
    })
}

func (ns *s3StorePlugin) DependencyReader() dependencystore.Reader {
	return ns.store
}

func (ns *s3StorePlugin) SpanReader() spanstore.Reader {
	return ns.store
}

func (ns *s3StorePlugin) SpanWriter() spanstore.Writer {
	return ns.store
}

func (ns *s3StorePlugin) ArchiveSpanReader() spanstore.Reader {
	return ns.archiveStore
}

func (ns *s3StorePlugin) ArchiveSpanWriter() spanstore.Writer {
	return ns.archiveStore
}

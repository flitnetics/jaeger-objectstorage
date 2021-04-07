package main

import (
    "flag"

    "github.com/jaegertracing/jaeger/plugin/storage/grpc"
    "github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
    "github.com/jaegertracing/jaeger/storage/dependencystore"
    "github.com/jaegertracing/jaeger/storage/spanstore"
)
    
type s3StorePlugin interface {
   	SpanReader() spanstore.Reader
   	SpanWriter() spanstore.Writer
   	DependencyReader() dependencystore.Reader
}

func readS3() {

}

func writeS3() {

}

func dependencyReader() {

}

func main() {
    var configPath string
    flag.StringVar(&configPath, "config", "", "A path to the plugin's configuration file")
    flag.Parse()

    plugin := s3StorePlugin{
            readS3(),
            writeS3(),
            dependencyReader(),
    }
    
    grpc.Serve(&shared.PluginServices{
        Store:        plugin,
    })
}

package main

import (
    "flag"

    "github.com/jaegertracing/jaeger/plugin/storage/grpc"
    "github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
    "github.com/jaegertracing/jaeger/storage/dependencystore"
    "github.com/jaegertracing/jaeger/storage/spanstore"
)
    
type StoragePlugin interface {
        SpanReader() spanstore.Reader
   	SpanWriter() spanstore.Writer
   	DependencyReader() dependencystore.Reader
}

func SpanReader() {

}

func SpanWriter() {

}

func DependencyReader() {

}

func main() {
    var configPath string
    flag.StringVar(&configPath, "config", "", "A path to the plugin's configuration file")
    flag.Parse()

    plugin := StoragePlugin{}
    
    grpc.Serve(&shared.PluginServices{
        Store:        plugin,
    })
}

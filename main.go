package main

import (
    "flag"
    "github.com/jaegertracing/jaeger/plugin/storage/grpc"
)
    
func main() {
    var configPath string
    flag.StringVar(&configPath, "config", "", "A path to the plugin's configuration file")
    flag.Parse()

    plugin := myStoragePlugin{}
    
    grpc.Serve(&shared.PluginServices{
        Store:        plugin,
	ArchiveStore: plugin,
    })
}

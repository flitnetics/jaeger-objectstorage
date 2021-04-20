This is the repository that contains S3 plugin for Jaeger.

> IMPORTANT: This plugin is still under development.
## Preresquities
* Need my own fork of jaeger

## Compile
Need to compile my own fork of jaeger (develop branch)
```
git clone -b develop git@github.com:muhammadn/jaeger.git
cd /path/to/jaeger
go build -tags ui ./cmd/all-in-one/main.go
```

In order to compile the plugin from source code you can use `go build`:

```
go build ./cmd/jaeger-s3/
```
## Start
In order to start plugin just tell jaeger the path to a config compiled plugin.

```
GRPC_STORAGE_PLUGIN_BINARY="./jaeger-s3" GRPC_STORAGE_PLUGIN_CONFIGURATION_FILE=./config-example.yaml SPAN_STORAGE_TYPE=grpc-plugin  GRPC_STORAGE_PLUGIN_LOG_LEVEL=DEBUG ./all-in-one --sampling.strategies-file=/location/of/your/jaeger/cmd/all-in-one/sampling_strategies.json
```

## License

The S3 Storage gRPC Plugin for Jaeger is an [MIT licensed](LICENSE) open source project.

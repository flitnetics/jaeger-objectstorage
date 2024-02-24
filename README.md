This is the repository that contains object storage connector for Jaeger.

## About
Jaeger connector to Tempo

**Version 3 of this plugin is not compatible with Version 2**

## Build/Compile
In order to compile the plugin from source code you can use `go build`:

```
cd /path/to/jaeger-objectstorage
go build ./cmd/jaeger-objectstorage
```

## Configuration
### Requirements
Use our fork of tempo [https://github.com/flitnetics/tempo](https://github.com/flitnetics/tempo)

## Backend Configuration
```config.yaml
backend: your.tempo.backend:3200
```

## Running Jaeger with Object Storage Plugin
In the same directory that you had compiled jaeger-objectstorage with `config.yaml` located, run:

```
docker run --name jaeger -it -e SPAN_STORAGE_TYPE=grpc-plugin \                             
  -e GRPC_STORAGE_PLUGIN_BINARY="/app/jaeger-tempo" \
  -e GRPC_STORAGE_PLUGIN_CONFIGURATION_FILE=/app/config.yml \
  -e GRPC_STORAGE_PLUGIN_LOG_LEVEL=DEBUG --mount type=bind,source="$(pwd)",target=/app \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411  \
  jaegertracing/all-in-one:1.54
```

## License

The Object Storage gRPC Plugin for Jaeger is an [Apache licensed](LICENSE) open source project.

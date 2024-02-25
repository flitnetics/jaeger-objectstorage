This is the repository that contains object storage connector for Jaeger.

For support questions, please go to [https://community.flitnetics.com](https://community.flitnetics.com)

## About
Jaeger connector to Tempo

**Version 3 of this plugin is not compatible with Version 2**


Version 3 is written with OTEL specifications to make it interoperate better.

## Build/Compile
In order to compile the plugin from source code you can use `go build`:

```
cd /path/to/jaeger-objectstorage
go build ./cmd/jaeger-objectstorage
```

## Configuration
### Requirements
Use our fork of tempo [https://github.com/flitnetics/tempo](https://github.com/flitnetics/tempo)

To make it simpler, we have built a docker image for our tempo fork:
[https://github.com/flitnetics/tempo/pkgs/container/tempo](https://github.com/flitnetics/tempo/pkgs/container/tempo)

Expose ports for tempo image's container: 3200,4317,4318

Explanation: 
  * tempo backend: 3200
  * otel grpc: 4317
  * otel http: 4318

We try to match to the same stable version from upstream tempo with our own modifications.

Example Tempo configuration for demo purposes:
```
server:
  http_listen_port: 3200

distributor:
  receivers:                           # this configuration will listen on all ports and protocols that tempo is capable of.
    jaeger:                            # the receives all come from the OpenTelemetry collector.  more configuration information can
      protocols:                       # be found there: https://github.com/open-telemetry/opentelemetry-collector/tree/main/receiver
        thrift_http:                   #
        grpc:                          # for a production deployment you should only enable the receivers you need!
        thrift_binary:
        thrift_compact:
    zipkin:
    otlp:
      protocols:
        http:
        grpc:
    opencensus:

ingester:
  max_block_duration: 5m               # cut the headblock when this much time passes. this is being set for demo purposes and should probably be left alone normally

compactor:
  compaction:
    block_retention: 1h                # overall Tempo trace retention. set for demo purposes

metrics_generator:
  registry:
    external_labels:
      source: tempo
      cluster: docker-compose
  storage:
    path: /tmp/tempo/generator/wal
    remote_write:
      - url: http://prometheus:9090/api/v1/write
        send_exemplars: true

storage:
  trace:
    backend: s3                        # backend configuration to use
    wal:
      path: /tmp/tempo/wal             # where to store the the wal locally
    s3:
      bucket: tempo                    # how to store data in s3
      endpoint: yours3endpoint
      access_key: s3accesskey
      secret_key: s3secret
      insecure: true
      # For using AWS, select the appropriate regional endpoint and region
      # endpoint: s3.dualstack.us-west-2.amazonaws.com
      # region: us-west-2

overrides:
  metrics_generator_processors: [service-graphs, span-metrics]
```
## Backend Configuration
This file is placed in your plugin's directory (see the next section to understand more).

```config.yaml
backend: your.tempo.backend:3200
```

## Running Jaeger with Object Storage Plugin
In the same directory that you had compiled jaeger-objectstorage with `config.yaml` located, run:

```
docker run --name jaeger -it -e SPAN_STORAGE_TYPE=grpc-plugin \                             
  -e GRPC_STORAGE_PLUGIN_BINARY="/app/jaeger-objectstorage" \
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

## Testing with Jaeger Example HotROD application
```
docker run \                                                                                                          2 ↵
  --rm \
  --link jaeger -e JAEGER_AGENT_HOST="jaeger" \
  --env OTEL_EXPORTER_OTLP_ENDPOINT=http://<tempo backend ip or host>:4318 \
  -p8080-8083:8080-8083 \
  jaegertracing/example-hotrod:latest \
  all
```
## License

The Object Storage gRPC Plugin for Jaeger is an [Apache licensed](LICENSE) open source project.

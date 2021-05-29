This is the repository that contains S3 plugin for Jaeger.

## About
S3 storage support for Jaeger. 

Google Cloud Storage (GCS), Microsoft Azure Blob Storage, Amazon DynamoDB and Google BigTable **may** work with some changes to configuration file. Reports on testing on these storage backends is appreciated.

## Preresquities
None. No longer needs my custom jaeger code. Just use the official ones.

You can now use this plugin with Jaeger Operator/Helmchart/K8s since there are no dependency on my custom Jaeger code.

## Build/Compile
In order to compile the plugin from source code you can use `go build`:

```
cd /path/to/jaeger-s3
go build ./cmd/jaeger-s3/
```

## Configuration
#### Storage
[https://github.com/grafana/loki/blob/37a7189d4ed76655144d982e2eeebf495e0809ea/docs/sources/configuration/_index.md#storage_config](https://github.com/grafana/loki/blob/37a7189d4ed76655144d982e2eeebf495e0809ea/docs/sources/configuration/_index.md#storage_config)
#### Index (schema config)
[https://github.com/grafana/loki/blob/37a7189d4ed76655144d982e2eeebf495e0809ea/docs/sources/configuration/_index.md#schema_config](https://github.com/grafana/loki/blob/37a7189d4ed76655144d982e2eeebf495e0809ea/docs/sources/configuration/_index.md#schema_config)
#### More info
[https://grafana.com/docs/loki/latest/operations/storage/boltdb-shipper/](https://grafana.com/docs/loki/latest/operations/storage/boltdb-shipper/)

Sample basic config:
```
schema_config:
  configs:
    - from: 2018-10-24
      store: boltdb-shipper
      object_store: s3
      schema: v11
      index:
        prefix: index_
        period: 24h
      row_shards: 10

storage_config:
  aws:
    bucketnames: bucketname
    region: ap-southeast-1
    access_key_id: aws_access_key_id
    secret_access_key: aws_secret_access_key
    endpoint: s3.ap-southeast-1.amazonaws.com
    http_config:
      idle_conn_timeout: 90s
      response_header_timeout: 0s
      tls_handshake_timeout: 3s # change this to something larger if you have `TLS Handshake Timeout` or 0 to disable timeout
  boltdb_shipper:
    active_index_directory: /tmp/loki/boltdb-shipper-active
    cache_location: /tmp/loki/boltdb-shipper-cache
    cache_ttl: 24h         # Can be increased for faster performance over longer query periods, uses more disk space
    shared_store: s3
  filesystem:
    directory: /tmp/loki/chunks

compactor:
  working_directory: /tmp/loki/boltdb-shipper-compactor
  shared_store: s3
```
## Start
In order to start plugin just tell jaeger the path to a config compiled plugin.

```
GRPC_STORAGE_PLUGIN_BINARY="./jaeger-s3" GRPC_STORAGE_PLUGIN_CONFIGURATION_FILE=./config-example.yaml SPAN_STORAGE_TYPE=grpc-plugin  GRPC_STORAGE_PLUGIN_LOG_LEVEL=DEBUG ./all-in-one --sampling.strategies-file=/location/of/your/jaeger/cmd/all-in-one/sampling_strategies.json
```

## License

The S3 Storage gRPC Plugin for Jaeger is an [MIT licensed](LICENSE) open source project.

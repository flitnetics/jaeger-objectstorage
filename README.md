This is the repository that contains object storage (S3/GCS/AzureBlob) plugin for Jaeger.

You are free to use this software under a permissive open-source MIT license.

To fund further work and maintenance on this plugin, work will be done by flitnetics.

If you require additional support for your infrastructure, you can contact [Sales](mailto:sales@flitnetics.com)

## About
S3, Google Cloud Storage(GCS) and Microsoft Azure Blob Storage object storage support for Jaeger. 

Amazon DynamoDB and Google BigTable for indexes should work as this code. 

The configuration is mostly identical on how you configure storage, indexes, compactors, rulers, etc in [Loki](https://github.com/grafana/loki).

With this plugin, you won't need to run and maintain Tempo, at all!

Works with [Kiali](https://kiali.io).

**Version 2 of this plugin is not compatible with Version 1**

## Build/Compile
In order to compile the plugin from source code you can use `go build`:

```
cd /path/to/jaeger-objectstorage
go build ./cmd/jaeger-objectstorage
```

## Configuration
#### Storage
[https://github.com/grafana/loki/blob/37a7189d4ed76655144d982e2eeebf495e0809ea/docs/sources/configuration/_index.md#storage_config](https://github.com/grafana/loki/blob/37a7189d4ed76655144d982e2eeebf495e0809ea/docs/sources/configuration/_index.md#storage_config)
#### Index (schema config)
[https://github.com/grafana/loki/blob/37a7189d4ed76655144d982e2eeebf495e0809ea/docs/sources/configuration/_index.md#schema_config](https://github.com/grafana/loki/blob/37a7189d4ed76655144d982e2eeebf495e0809ea/docs/sources/configuration/_index.md#schema_config)
#### More info
[https://grafana.com/docs/loki/latest/operations/storage/boltdb-shipper/](https://grafana.com/docs/loki/latest/operations/storage/boltdb-shipper/)

Sample basic config (AWS):
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
      row_shards: 32

storage_config:
  aws:
    region: ap-southeast-1
    access_key_id: aws_access_key_id
    secret_access_key: aws_secret_access_key
    endpoint: s3.ap-southeast-1.amazonaws.com
    http_config:
      idle_conn_timeout: 90s
      response_header_timeout: 0s
  boltdb_shipper:
    active_index_directory: /tmp/loki/boltdb-shipper-active
    cache_location: /tmp/loki/boltdb-shipper-cache
    cache_ttl: 24h
    shared_store: s3
  filesystem:
    directory: /tmp/loki/chunks

compactor:
  working_directory: /tmp/loki/boltdb-shipper-compactor
  shared_store: s3

limits_config:
  enforce_metric_name: false
  reject_old_samples: true
  reject_old_samples_max_age: 168h

memberlist:
  abort_if_cluster_join_fails: false

  max_join_backoff: 1m
  max_join_retries: 10
  min_join_backoff: 1s

distributor:
  ring:
    kvstore:
      store: memberlist

ingester:
  lifecycler:
    ring:
      kvstore:
        store: memberlist
      replication_factor: 1
    final_sleep: 0s
  chunk_idle_period: 5m
  chunk_retain_period: 30s
```

Changes for AWS with Retention for 28 days
**(Update and add the necessary bits like above, ie. "compactor" and "table_manager")**
```
compactor:
  working_directory: /tmp/loki/boltdb-shipper-compactor
  shared_store: s3
  compaction_interval: 10m
  retention_enabled: true
  retention_delete_delay: 2h
  retention_delete_worker_count: 150

table_manager:
  retention_deletes_enabled: true
  retention_period: 672h
```

Sample basic config (AWS with IRSA. example: EKS)

**replace aws part of configuration above with URI style instead**
```
storage_config:
  aws:
    s3: s3://ap-southeast-1/yourbucket
    http_config:
      idle_conn_timeout: 90s
      response_header_timeout: 0s
```

Sample basic config (GCS):
**replace the necessary parts from the "Sample basic config (AWS)" above**
```
storage_config:
  boltdb_shipper:
    active_index_directory: /loki/boltdb-shipper-active
    cache_location: /loki/boltdb-shipper-cache
    cache_ttl: 24h         # Can be increased for faster performance over longer query periods, uses more disk space
    shared_store: gcs
  gcs:
      bucket_name: <bucket>

schema_config:
  configs:
    - from: 2020-07-01
      store: boltdb-shipper
      object_store: gcs
      schema: v11
      index:
        prefix: index_
        period: 24h
      row_shards: 32

compactor:
  working_directory: /tmp/loki/boltdb-shipper-compactor
  shared_store: gcs
```

Sample basic config (Azure BlobStorage):
**replace the necessary parts from the "Sample basic config (AWS)" above**
```
schema_config:
  configs:
    - from: 2020-07-01
      store: boltdb-shipper
      object_store: azure
      schema: v11  
      index:
        prefix: index_
        period: 24h
      row_shards: 32

storage_config:
  boltdb_shipper:
    active_index_directory: /data/loki/index
    shared_store: azure
    cache_location: /data/loki/boltdb-cache
  azure:
    container_name: .. # add container name here
    account_name: .. # add account name here
    account_key: .. # add access key here

compactor:
  working_directory: /tmp/loki/boltdb-shipper-compactor
  shared_store: azure
```

## Start
In order to start plugin just tell jaeger the path to a config compiled plugin.

```
GRPC_STORAGE_PLUGIN_BINARY="./jaeger-objectstorage" GRPC_STORAGE_PLUGIN_CONFIGURATION_FILE=./config-example.yaml SPAN_STORAGE_TYPE=grpc-plugin  GRPC_STORAGE_PLUGIN_LOG_LEVEL=DEBUG ./all-in-one --sampling.strategies-file=/location/of/your/jaeger/cmd/all-in-one/sampling_strategies.json
```

For Jaeger Operator on Kubernetes for testing/demo **!!NOT PRODUCTION!!**, sample manifest:

```
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: observability

commonLabels:
  app.kubernetes.io/instance: observability

resources:
  - https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/crds/jaegertracing.io_jaegers_crd.yaml
  - https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/service_account.yaml
  - https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role.yaml
  - https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/role_binding.yaml
  - https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/operator.yaml
  - https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/cluster_role.yaml
  - https://raw.githubusercontent.com/jaegertracing/jaeger-operator/master/deploy/cluster_role_binding.yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger-operator
  app: jaeger
spec:
  template:
    spec:
      containers:
      - name: jaeger-operator
        image: jaegertracing/jaeger-operator:master
        args: ["start"]
        env:
        - name: LOG-LEVEL
          value: debug
---
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: jaeger-objectstorage
spec:
  strategy: allInOne
  allInOne:
    image: jaegertracing/all-in-one:latest
    options:
      log-level: debug
  storage:
    type: grpc-plugin
    grpcPlugin:
      image: ghcr.io/muhammadn/jaeger-objectstorage:latest
    options:
      grpc-storage-plugin:
        binary: /plugin/jaeger-objectstorage
        configuration-file: /plugin-config/config-example.yaml
        log-level: debug
  volumeMounts:
    - name: config-volume
      mountPath: /plugin-config
  volumes:
    - name: config-volume
      configMap:
        name: jaeger-objectstorage-config
---
apiVersion: v1
data:
  config-example.yaml: |-
    schema_config:
      configs:
        - from: 2018-10-24
          store: boltdb-shipper
          object_store: s3
          schema: v11
          index:
            prefix: index_
            period: 24h
          row_shards: 32

    storage_config:
      aws:
        bucketnames: yourbuckethere
        region: ap-southeast-1
        access_key_id: youraccesskey
        secret_access_key: youraccesssecret
        endpoint: s3.ap-southeast-1.amazonaws.com
        http_config:
          idle_conn_timeout: 90s
          response_header_timeout: 0s
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

    limits_config:
      enforce_metric_name: false
      reject_old_samples: true
      reject_old_samples_max_age: 168h

    memberlist:
      abort_if_cluster_join_fails: false

      max_join_backoff: 1m
      max_join_retries: 10
      min_join_backoff: 1s

    distributor:
      ring:
        kvstore:
          store: memberlist

    ingester:
      lifecycler:
        ring:
          kvstore:
            store: memberlist
          replication_factor: 1
        final_sleep: 0s
      chunk_idle_period: 5m
      chunk_retain_period: 30s
kind: ConfigMap
metadata:
  name: jaeger-objectstorage-config
```

## License

The S3 Storage gRPC Plugin for Jaeger is an [MIT licensed](LICENSE) open source project.

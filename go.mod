module jaeger-s3

go 1.16

require (
	github.com/Azure/azure-pipeline-go v0.2.2 // indirect
	github.com/Masterminds/sprig/v3 v3.2.2 // indirect
	github.com/alicebob/gopher-json v0.0.0-20200520072559-a9ecdc9d1d3a // indirect
	github.com/cortexproject/cortex v1.9.1-0.20210527130655-bd720c688ffa
	github.com/grafana/loki v1.6.1
	github.com/hashicorp/go-hclog v0.15.0
	github.com/jaegertracing/jaeger v1.17.1
	github.com/muhammadn/loki v1.7.0 // indirect
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/prometheus/common v0.23.0
	github.com/prometheus/prometheus v1.8.2-0.20210510213326-e313ffa8abf6
	github.com/sercand/kuberesolver v2.4.0+incompatible // indirect
	github.com/spf13/viper v1.7.0
	github.com/weaveworks/common v0.0.0-20210419092856-009d1eebd624
)

replace github.com/hpcloud/tail => github.com/grafana/tail v0.0.0-20201004203643-7aa4e4a91f03

replace github.com/Azure/azure-sdk-for-go => github.com/Azure/azure-sdk-for-go v36.2.0+incompatible

// Keeping this same as Cortex to avoid dependency issues.
replace k8s.io/client-go => k8s.io/client-go v0.19.4

replace k8s.io/api => k8s.io/api v0.19.4

replace github.com/hashicorp/consul => github.com/hashicorp/consul v1.5.1

// >v1.2.0 has some conflict with prometheus/alertmanager. Hence prevent the upgrade till it's fixed.
replace github.com/satori/go.uuid => github.com/satori/go.uuid v1.2.0

// Use fork of gocql that has gokit logs and Prometheus metrics.
replace github.com/gocql/gocql => github.com/grafana/gocql v0.0.0-20200605141915-ba5dc39ece85

// Same as Cortex, we can't upgrade to grpc 1.30.0 until go.etcd.io/etcd will support it.
replace google.golang.org/grpc => google.golang.org/grpc v1.29.1

// Same as Cortex
// Using a 3rd-party branch for custom dialer - see https://github.com/bradfitz/gomemcache/pull/86
replace github.com/bradfitz/gomemcache => github.com/themihai/gomemcache v0.0.0-20180902122335-24332e2d58ab

// Fix errors like too many arguments in call to "github.com/go-openapi/errors".Required
//   have (string, string)
//   want (string, string, interface {})
replace github.com/go-openapi/errors => github.com/go-openapi/errors v0.19.4

replace github.com/go-openapi/validate => github.com/go-openapi/validate v0.19.8

replace github.com/grafana/loki v1.6.1 => github.com/muhammadn/loki v1.6.11

replace github.com/cortexproject/cortex => github.com/muhammadn/cortex v1.8.9

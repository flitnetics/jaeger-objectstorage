package s3store

import (
	"context"
	"time"
        "log"
        "path"
        "io/ioutil"
        "fmt"
        "strconv"

	"github.com/go-pg/pg/v9"
        "github.com/weaveworks/common/user"

	hclog "github.com/hashicorp/go-hclog"
        pmodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"

        "github.com/cortexproject/cortex/pkg/util/flagext"
        "github.com/cortexproject/cortex/pkg/chunk/storage"
        util_log "github.com/cortexproject/cortex/pkg/util/log"
        cortex_local "github.com/cortexproject/cortex/pkg/chunk/local"
        "github.com/cortexproject/cortex/pkg/chunk"

        lstore "jaeger-s3/storage"
        "github.com/grafana/loki/pkg/util/validation"
	"github.com/grafana/loki/pkg/logql"

        "jaeger-s3/config/types"
)

var _ spanstore.Reader = (*Reader)(nil)

var (
        userCtx        = user.InjectOrgID(context.Background(), "data")
)

// Reader can query for and load traces from PostgreSQL v2.x.
type Reader struct {
	db *pg.DB
        cfg    *types.Config

        store  lstore.Store
	logger hclog.Logger
}

// NewReader returns a new SpanReader for PostgreSQL v2.x.
func NewReader(db *pg.DB, cfg *types.Config, store lstore.Store, logger hclog.Logger) *Reader {
	return &Reader{
                cfg: cfg,
		db:     db,
                store:  store,
		logger: logger,
	}
}

// GetServices returns all services traced by Jaeger
func (r *Reader) GetServices(ctx context.Context) ([]string, error) {
	r.logger.Warn("GetServices called")

        //var fooLabelsWithName = "{__name__=\"service\", env=\"prod\"}"
        var fooLabelsWithName = "{env=\"prod\", __name__=\"services\"}"

        chunks, err := r.store.Get(userCtx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(fooLabelsWithName)...)
        //log.Println("rchunk get: %s", chunks)
        /* for i := 0; i < len(chunks); i++ {
                log.Println(chunks[i].Metric[8].Value)
        } */

        ret := removeServiceDuplicateValues(chunks, "service_name")

        return ret, err
}

func removeServiceDuplicateValues(a []chunk.Chunk, b string) []string {
    keys := make(map[string]bool)
    list := []string{}
 
    // If the key(values of the slice) is not equal
    // to the already present value in new slice (list)
    // then we append it. else we jump on another element.
    for _, entry := range a {
        if _, value := keys[entry.Metric[8].Value]; !value {
            // data type: service_name, operation_name, etc
            if entry.Metric[8].Name == b {
                    // assign key value to list
                    keys[entry.Metric[8].Value] = true
                    list = append(list, entry.Metric[8].Value)
            }
        }
    }
    return list
}

func removeOperationDuplicateValues(a []chunk.Chunk, b string) []string {
    keys := make(map[string]bool)
    list := []string{}

    // If the key(values of the slice) is not equal
    // to the already present value in new slice (list)
    // then we append it. else we jump on another element.
    for _, entry := range a {
        if _, value := keys[entry.Metric[5].Value]; !value {
            // data type: service_name, operation_name, etc
            if entry.Metric[5].Name == b {
                    // assign key value to list
                    keys[entry.Metric[5].Value] = true
                    list = append(list, entry.Metric[5].Value)
            }
        }
    }
    return list
}

// GetOperations returns all operations for a specific service traced by Jaeger
func (r *Reader) GetOperations(ctx context.Context, param spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {

        tempDir, err := ioutil.TempDir("", "boltdb-shippers")
        if err != nil {
                log.Println("tempDir failure %s", err)
        }

        limits, err := validation.NewOverrides(validation.Limits{}, nil)
        if err != nil {
                log.Println("limits failure %s", err)
        }

        // config for BoltDB Shipper
        boltdbShipperConfig := r.cfg.StorageConfig.BoltDBShipperConfig
        flagext.DefaultValues(&boltdbShipperConfig)

        kconfig := &lstore.Config{
                Config: storage.Config{
                        AWSStorageConfig: r.cfg.StorageConfig.AWSStorageConfig,
                        FSConfig: cortex_local.FSConfig{Directory: path.Join(tempDir, "chunks")},
                },
                BoltDBShipperConfig: boltdbShipperConfig,
        }

        rChunkStore, err := storage.NewStore(
                kconfig.Config,
                r.cfg.ChunkStoreConfig,
                r.cfg.SchemaConfig.SchemaConfig,
                limits,
                nil,
                nil,
                util_log.Logger,
        )

        //var fooLabelsWithName = "{__name__=\"service\", env=\"prod\"}"
        var fooLabelsWithName = "{env=\"prod\", __name__=\"services\"}"

        if rChunkStore != nil {
                rstore, err := lstore.NewStore(*kconfig, r.cfg.SchemaConfig, rChunkStore, nil)
                if err != nil {
                       log.Println("read store error: %s", err)
                }

                chunks, err := rstore.Get(userCtx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(fooLabelsWithName)...)
                operations := removeOperationDuplicateValues(chunks, "operation_name")

                ret := make([]spanstore.Operation, 0, len(operations))
                for _, operation := range operations {
                        if len(operation) > 0 {
                                ret = append(ret, spanstore.Operation{Name: operation})
                        }
                }

                return ret, err
        }

        return nil, err
}

// GetTrace takes a traceID and returns a Trace associated with that traceID
func (r *Reader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	builder := &whereBuilder{where: "", params: make([]interface{}, 0)}
        log.Println("GetTrace executed")

	if traceID.Low > 0 {
		builder.andWhere(traceID.Low, "trace_id_low = ?")
	}
	if traceID.High > 0 {
		builder.andWhere(traceID.Low, "trace_id_high = ?")
	}

        var fooLabelsWithName = fmt.Sprintf("{env=\"prod\", __name__=\"services\", trace_id_low=\"%s\", trace_id_high=\"%s\"}", traceID.Low, traceID.Low)

        chunks, err := r.store.Get(userCtx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(fooLabelsWithName)...)

        var spans []Span
        ret := make([]*model.Span, 0, len(spans))
        ret2 := make([]model.Trace_ProcessMapping, 0, len(spans))
        for _, chunk := range chunks {
                var serviceName string
                var processId string
                var processTags map[string]interface{}

                if chunk.Metric[8].Name == "service_name" {
                        serviceName = chunk.Metric[8].Value
                }

                if chunk.Metric[6].Name == "process_id" {
                        processId = chunk.Metric[6].Value
                }

                if chunk.Metric[7].Name == "process_tags" {
                        processTags = StrToMap(chunk.Metric[7].Value)
                }

                ret = append(ret, toModelSpan(chunk))
                ret2 = append(ret2, model.Trace_ProcessMapping{
                        ProcessID: processId,
                        Process: model.Process{
                                ServiceName: serviceName,
                                Tags:        mapToModelKV(processTags),
                        },
                })
        }

	return &model.Trace{Spans: ret, ProcessMap: ret2}, err
}

func buildTraceWhere(query *spanstore.TraceQueryParameters) string { 
        log.Println("buildTraceWhere executed")
        //var builder map[int]interface{}
        var builder string

        builder = "{"
        builder = builder + "__name__=\"services\", env=\"prod\", "

	if len(query.ServiceName) > 0 {
                builder = builder + fmt.Sprintf("service_name = \"%s\", ", query.ServiceName)
	}
	if len(query.OperationName) > 0 {
                builder = builder + fmt.Sprintf("operation_name = \"%s\", ", query.OperationName)
	}
	//if query.StartTimeMin.After(time.Time{}) {
        //        builder = builder + fmt.Sprintf("start_time > \"%s\", ", query.StartTimeMin)
	//}
	if query.StartTimeMax.After(time.Time{}) {
		//TODO builder.andWhere(query.StartTimeMax, "start_time < ?")
	}
	if query.DurationMin > 0*time.Second {
                builder = builder + fmt.Sprintf("duration < \"%d\", ", query.DurationMin)
	}
	if query.DurationMax > 0*time.Second {
                builder = builder + fmt.Sprintf("duration > \"%d\"", query.DurationMax)
	}
	//TODO Tags map[]string
        // Remove last two characters (space and comma)
        builder = builder[:len(builder)-2]
        builder = builder + "}"

        log.Println("builder: %s", builder)

	return builder
}

// FindTraces retrieve traces that match the traceQuery
func (r *Reader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
       log.Println("FindTraces executed")

       traceIDs, err := r.FindTraceIDs(ctx, query)
       ret := make([]*model.Trace, 0, len(traceIDs))
       if err != nil {
               return ret, err
       }
       grouping := make(map[model.TraceID]*model.Trace)
       //idsLow := make([]uint64, 0, len(traceIDs))
       for _, traceID := range traceIDs {
               //idsLow = append(idsLow, traceID.Low)
               var fooLabelsWithName = fmt.Sprintf("{env=\"prod\", __name__=\"services\", trace_id_low=\"%d\"}", traceID.Low)

               if err != nil {
                       log.Println("read store error: %s", err)
               }

               chunks, err := r.store.Get(userCtx, "data", timeToModelTime(time.Now().Add(-30 * time.Minute)), timeToModelTime(time.Now()), newMatchers(fooLabelsWithName)...)
               if err != nil {
                       log.Println("Error getting data in reader: %s", err)
               }
               for _, chunk := range chunks {
                       var serviceName string
                       var processId string
                       var processTags map[string]interface{}

                       if chunk.Metric[8].Name == "service_name" {
                                serviceName = chunk.Metric[8].Value
                       }
                
                       if chunk.Metric[6].Name == "process_id" {
                                processId = chunk.Metric[6].Value
                       }
                
                       if chunk.Metric[7].Name == "process_tags" {
                                processTags = StrToMap(chunk.Metric[7].Value)
                       }

                       modelSpan := toModelSpan(chunk)
                       trace, found := grouping[modelSpan.TraceID]
                       if !found {
                               trace = &model.Trace{
                                       Spans:      make([]*model.Span, 0, len(chunks)),
                                       ProcessMap: make([]model.Trace_ProcessMapping, 0, len(chunks)),
                               }
                               grouping[modelSpan.TraceID] = trace
                       }
                       trace.Spans = append(trace.Spans, modelSpan)
                       procMap := model.Trace_ProcessMapping{
                               ProcessID: processId,
                               Process: model.Process{
                                       ServiceName: serviceName,
                                       Tags:        mapToModelKV(processTags),
                               },
                       }
                       trace.ProcessMap = append(trace.ProcessMap, procMap)
               }
       }

       for _, trace := range grouping {
               ret = append(ret, trace)
       }

       return ret, err
}

// FindTraceIDs retrieve traceIDs that match the traceQuery
func (r *Reader) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) (ret []model.TraceID, err error) {
	builder := buildTraceWhere(query)

        tempDir, err := ioutil.TempDir("", "boltdb-shippers")
        if err != nil {
                log.Println("tempDir failure %s", err)
        }

        limits, err := validation.NewOverrides(validation.Limits{}, nil)
        if err != nil {
                log.Println("limits failure %s", err)
        }

        // config for BoltDB Shipper
        boltdbShipperConfig := r.cfg.StorageConfig.BoltDBShipperConfig
        flagext.DefaultValues(&boltdbShipperConfig)

        kconfig := &lstore.Config{
                Config: storage.Config{
                        AWSStorageConfig: r.cfg.StorageConfig.AWSStorageConfig,
                        FSConfig: cortex_local.FSConfig{Directory: path.Join(tempDir, "chunks")},
                },
                BoltDBShipperConfig: boltdbShipperConfig,
        }

        rChunkStore, err := storage.NewStore(
                kconfig.Config,
                r.cfg.ChunkStoreConfig,
                r.cfg.SchemaConfig.SchemaConfig,
                limits,
                nil,
                nil,
                util_log.Logger,
        )

        var fooLabelsWithName = builder

        if rChunkStore != nil {
                rstore, err := lstore.NewStore(*kconfig, r.cfg.SchemaConfig, rChunkStore, nil)
                if err != nil {
                       log.Println("read store error: %s", err)
                }

                ret, err := rstore.Get(userCtx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(fooLabelsWithName)...)
                if err != nil {
                        log.Println("rstore error: %s", err)
                }
 
                var trace model.TraceID
                var traces []model.TraceID
                for i := 0; i < len(ret); i++ {
                        if ret[i].Metric[11].Name == "trace_id_low" {
                                low, _ := strconv.ParseUint(ret[i].Metric[11].Value, 10, 64)
                                trace.Low = low
                        }
                        if ret[i].Metric[11].Name == "trace_id_high" {
                                high, _ := strconv.ParseUint(ret[i].Metric[11].Value, 10, 64)
                                trace.High = high
                        }
                        traces = append(traces, trace) 
                }
 
	       return traces, err
        }
        return nil, err
}

// GetDependencies returns all inter-service dependencies
func (r *Reader) GetDependencies(endTs time.Time, lookback time.Duration) (ret []model.DependencyLink, err error) {
	err = r.db.Model((*SpanRef)(nil)).
		ColumnExpr("source_spans.service_name as parent").
		ColumnExpr("child_spans.service_name as child").
		ColumnExpr("count(*) as call_count").
		Join("JOIN spans AS source_spans ON source_spans.id = span_ref.source_span_id").
		Join("JOIN spans AS child_spans ON child_spans.id = span_ref.child_span_id").
		Group("source_spans.service_name").
		Group("child_spans.service_name").
		Select(&ret)

	return ret, err
}

func timeToModelTime(t time.Time) pmodel.Time {
	return pmodel.TimeFromUnixNano(t.UnixNano())
}

func newMatchers(matchers string) []*labels.Matcher {
	res, err := logql.ParseMatchers(matchers)
	if err != nil {
		log.Println("parseMatchers: %s", err)
	}
	return res
}

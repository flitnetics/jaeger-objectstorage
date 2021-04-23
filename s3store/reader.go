package s3store

import (
	"context"
	"time"
        "log"
        "fmt"
        "strconv"

        "github.com/weaveworks/common/user"

	hclog "github.com/hashicorp/go-hclog"
        pmodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"

        "github.com/cortexproject/cortex/pkg/chunk"

        lstore "github.com/grafana/loki/pkg/storage"
	"github.com/grafana/loki/pkg/logql"

        "jaeger-s3/config/types"
)

var _ spanstore.Reader = (*Reader)(nil)

var (
        userCtx        = user.InjectOrgID(context.Background(), "data")
)

// Reader can query for and load traces from PostgreSQL v2.x.
type Reader struct {
        cfg    *types.Config

        store  lstore.Store
	logger hclog.Logger
}

// NewReader returns a new SpanReader for PostgreSQL v2.x.
func NewReader(cfg *types.Config, store lstore.Store, logger hclog.Logger) *Reader {
	return &Reader{
                cfg: cfg,
                store:  store,
		logger: logger,
	}
}

// GetServices returns all services traced by Jaeger
func (r *Reader) GetServices(ctx context.Context) ([]string, error) {
	r.logger.Warn("GetServices called")

        //var fooLabelsWithName = "{__name__=\"service\", env=\"prod\"}"
        var fooLabelsWithName = "{env=\"prod\", __name__=\"spans\"}"

        chunks, err := r.store.Get(userCtx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(fooLabelsWithName)...)
        //log.Println("chunks get: %s", chunks)
        /* for i := 0; i < len(chunks); i++ {
                log.Println(chunks[i].Metric[9].Value)
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

        //var fooLabelsWithName = "{__name__=\"service\", env=\"prod\"}"
        var fooLabelsWithName = "{env=\"prod\", __name__=\"spans\"}"

        chunks, err := r.store.Get(userCtx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(fooLabelsWithName)...)
        operations := removeOperationDuplicateValues(chunks, "operation_name")

        ret := make([]spanstore.Operation, 0, len(operations))
        for _, operation := range operations {
                if len(operation) > 0 {
                        ret = append(ret, spanstore.Operation{Name: operation})
                }
        }

        return ret, err
}

// GetTrace takes a traceID and returns a Trace associated with that traceID
func (r *Reader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
        log.Println("GetTrace executed")

        var fooLabelsWithName = fmt.Sprintf("{env=\"prod\", __name__=\"spans\", trace_id_low=\"%s\", trace_id_high=\"%s\"}", traceID.Low, traceID.Low)

        chunks, err := r.store.Get(userCtx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(fooLabelsWithName)...)

        ret := make([]*model.Span, 0, len(chunks))
        ret2 := make([]model.Trace_ProcessMapping, 0, len(chunks))
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

func buildTraceWhere(query *spanstore.TraceQueryParameters) (string, time.Time, time.Time) { 
        log.Println("buildTraceWhere executed")
        var builder string
        //log.Println("min time: %s", query.StartTimeMin)

        builder = "{"
        builder = builder + "__name__=\"spans\", env=\"prod\", "

	if len(query.ServiceName) > 0 {
                builder = builder + fmt.Sprintf("service_name = \"%s\", ", query.ServiceName)
	}
	if len(query.OperationName) > 0 {
                builder = builder + fmt.Sprintf("operation_name = \"%s\", ", query.OperationName)
	}
        if len(query.Tags) > 0 {
                for i, v := range query.Tags { 
                        builder = builder + fmt.Sprintf("tags =~ \".*%s:%s.*\", ", i, v)
                }
        }

        // Remove last two characters (space and comma)
        builder = builder[:len(builder)-2]
        builder = builder + "}"

        // filters
        if query.DurationMin > 0*time.Second {
                builder = builder + fmt.Sprintf(" | duration > %s", time.Duration(query.DurationMin) / time.Nanosecond)
        }
        if query.DurationMax > 0*time.Second {
                builder = builder + fmt.Sprintf(" | duration < %s", time.Duration(query.DurationMax) / time.Nanosecond)
        }

        // log our queries
        log.Println("builder: %s", builder)

        // here we include starttime min and max to pass to indexed timestamp
	return builder, query.StartTimeMin, query.StartTimeMax
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
       for _, traceID := range traceIDs {
               var fooLabelsWithName = fmt.Sprintf("{env=\"prod\", __name__=\"spans\", trace_id_low=\"%d\"}", traceID.Low)

               chunks, err := r.store.Get(userCtx, "data", timeToModelTime(time.Now().Add(-24 * time.Hour)), timeToModelTime(time.Now()), newMatchers(fooLabelsWithName)...)
               // log.Println("FindTraces chunks %s", chunks)
               //log.Println("traceID data %s", chunks)

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
	builder, timeMin, timeMax := buildTraceWhere(query)

        var fooLabelsWithName = builder

        chunks, err := r.store.Get(userCtx, "data", timeToModelTime(timeMin), timeToModelTime(timeMax), newMatchers(fooLabelsWithName)...)
        if err != nil {
                log.Println("store error: %s", err)
        }
 
        var trace model.TraceID
        var traces []model.TraceID
        for i := 0; i < len(chunks); i++ {
                if chunks[i].Metric[12].Name == "trace_id_low" {
                        low, _ := strconv.ParseUint(chunks[i].Metric[12].Value, 10, 64)
                        trace.Low = low
                }
                if chunks[i].Metric[11].Name == "trace_id_high" {
                        high, _ := strconv.ParseUint(chunks[i].Metric[11].Value, 10, 64)
                        trace.High = high
                }
                traces = append(traces, trace) 
        }
 
	return traces, err
}

// GetDependencies returns all inter-service dependencies
func (r *Reader) GetDependencies(endTs time.Time, lookback time.Duration) (ret []model.DependencyLink, err error) {
	/* err = r.db.Model((*SpanRef)(nil)).
		ColumnExpr("source_spans.service_name as parent").
		ColumnExpr("child_spans.service_name as child").
		ColumnExpr("count(*) as call_count").
		Join("JOIN spans AS source_spans ON source_spans.id = span_ref.source_span_id").
		Join("JOIN spans AS child_spans ON child_spans.id = span_ref.child_span_id").
		Group("source_spans.service_name").
		Group("child_spans.service_name").
		Select(&ret)

	return ret, err */
        return nil, err
}

func timeToModelTime(t time.Time) pmodel.Time {
	return pmodel.TimeFromUnixNano(t.UnixNano())
}

func newMatchers(matchers string) []*labels.Matcher {
        res, err := logql.ParseLogSelector(matchers, true)
        if err != nil {
                log.Println("parseLogSelector: %s", err)
        }
        return res.Matchers()
}

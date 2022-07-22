package loki

import (
	"context"
	"time"
        "log"
        "fmt"
        _ "strconv"
        "encoding/json"
	"github.com/go-logfmt/logfmt"
        "strings"
        "net/http"
        "net/url"
	"io/ioutil"
        "strconv"

        "github.com/weaveworks/common/user"

        hclog "github.com/hashicorp/go-hclog"

        pmodel "github.com/prometheus/common/model"
        "github.com/prometheus/prometheus/model/labels"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"

        lstore "github.com/grafana/loki/pkg/storage"
	"github.com/grafana/loki/pkg/logql"
)

var _ spanstore.Reader = (*Reader)(nil)

var (
        userCtx        = user.InjectOrgID(context.Background(), "fake")
)

// Reader can query for and load traces from your object store.
type Reader struct {
        cfg    Config
        store  lstore.Store
        logger hclog.Logger
}

type SpanData struct {
        // defining struct variables
        Duration      string  `json:"duration"`
        Env           string  `json:"env"`
        Flags         string  `json:"flags"`
        Id            string  `json:"id"`
        OperationName string  `json:"operation_name"`
        ProcessTags   string  `json:"process_tags"`
        ProcessId     string  `json:"process_id"`
        ServiceName   string  `json:"service_name"`
        StartTime     string  `json:"start_time"`
        Tags          string  `json:"tags"`
        TraceIdHigh   string  `json:"trace_id_high"`
        TraceIdLow    string  `json:"trace_id_low"`
}

type LokiStream struct {
        SpanData SpanData        `json:"stream"` 
        SValues  [][]string      `json:"values"`
}

type LokiResult struct {
        Stream []LokiStream `json:"result"`         
}

type LokiData struct {
        Result LokiResult `json:"data"`
}

// Span (not spanrange query)
type Metric struct {
        ServiceName   string  `json:"service_name"`
        OperationName string  `json:"operation_name"`
        Env           string  `json:"env"`
}

type sLokiStream struct {
        Metric   Metric `json:"metric"`
}

type sLokiResult struct {
        Stream []sLokiStream `json:"result"`
}

type sLokiData struct {
        Result sLokiResult `json:"data"`
}

// NewReader returns a new SpanReader for the object store.
func NewReader(cfg Config, store lstore.Store, logger hclog.Logger) *Reader {
	return &Reader{
                cfg: cfg,
                store:  store,
                logger: logger,
	}
}

func GetSpansRange(r *Reader, fooLabelsWithName string, startTime time.Time, endTime time.Time, resultsLimit uint32) (LokiData, error) {
         /* var s_labels LokiData

         params := logql.NewLiteralParams(
                 fooLabelsWithName,
                 startTime,
                 endTime,
                 0,
                 0,
                 logproto.BACKWARD,
                 resultsLimit,
                 nil,
        )

        qry := engine.Query(params)
        log.Println("Running GetSpans Query")

        // this is a tricky one, first we get ResultValue
        result, err := qry.Exec(userCtx)

       value, err := marshal.NewResultValue(result.Data)
       if err != nil {
                log.Println("Cannot convert it to ResultValue")
                 return LokiData{}, err
       }
 
        // add it to queryresponse
       q := loghttp.QueryResponse{
               Status: "success",
               Data: loghttp.QueryResponseData{
                       ResultType: value.Type(),
                       Result:     value,
                       Statistics: result.Statistics,
               },
       }

        // then convert it to JSON
        b, err := json.Marshal(q)
        if err != nil {
                 log.Println("Cannot marshal to json: %s", err)
        }

        bytes := []byte(b)

        //log.Println("JSON value for reader: %s", string(b))
        // then convert BACK to struct values from JSON
        err = json.Unmarshal(bytes, &s_labels)
        if err != nil {
               log.Println("Problem with unmarshalling json: %s", err)
        } 

        return s_labels, err */

        var s_labels LokiData

        query   := url.QueryEscape(fooLabelsWithName)
        httpurl := fmt.Sprintf("http://localhost:4100/loki/api/v1/query_range?direction=BACKWARD&limit=%d&query=%s&start=%d&end=%d", resultsLimit, query, startTime.UnixNano(), endTime.UnixNano())

        response, err := http.Get(httpurl)
        if err != nil {
                return LokiData{}, err
        }

        body, err := ioutil.ReadAll(response.Body)
        if err != nil {
                return LokiData{}, err
        }

        err = json.Unmarshal(body, &s_labels)
        if err != nil {
               log.Println("Problem with unmarshalling json: %s", err)
        }

        return s_labels, err
}

func GetSpans(limit uint32, fooLabelsWithName string) (sLokiData, error) {
        var s_labels sLokiData

        query   := url.QueryEscape(fooLabelsWithName)
        httpurl := fmt.Sprintf("http://localhost:4100/loki/api/v1/query?query=%s&time=%d&limit=%d", query, time.Now().UnixNano(), limit)

        response, err := http.Get(httpurl)
        if err != nil {
                return sLokiData{}, err
        }

        body, err := ioutil.ReadAll(response.Body)
        if err != nil {
                return sLokiData{}, err
        }

        err = json.Unmarshal(body, &s_labels)
        if err != nil {
               log.Println("Problem with unmarshalling json: %s", err)
        }

        return s_labels, err
}

// GetServices returns all services traced by Jaeger
func (r *Reader) GetServices(ctx context.Context) ([]string, error) {
	r.logger.Warn("GetServices called")

        //var fooLabelsWithName = "{env=\"prod\"}"
        var fooLabelsWithName = "count_over_time({env=\"prod\"}[1h])"

        // get the chunks
        //chunks, err := GetSpansRange(r, fooLabelsWithName, time.Now().Add(time.Duration(-1) * time.Hour), time.Now(), uint32(10000))
        chunks, err := GetSpans(uint32(1000), fooLabelsWithName)
 
        // clean up duplicates
        //ret := removeServiceDuplicateValues(chunks)

        spans := extractServices(chunks)
 
        return spans, err
}

func extractServices(a sLokiData) []string {
    list := []string{}
    keys := make(map[string]bool)

    for _, entry := range a.Result.Stream {
        if _, value := keys[entry.Metric.ServiceName]; !value {
                // assign key value to list
                keys[entry.Metric.ServiceName] = true
                list = append(list, entry.Metric.ServiceName)
        }
    }
    return list
}

func extractOperations(a sLokiData) []string {
    list := []string{}
    keys := make(map[string]bool)
    
    for _, entry := range a.Result.Stream {
        if _, value := keys[entry.Metric.OperationName]; !value {
                // assign key value to list
                keys[entry.Metric.ServiceName] = true
                list = append(list, entry.Metric.OperationName)
        }
    }
    return list
}

// GetOperations returns all operations for a specific service traced by Jaeger
func (r *Reader) GetOperations(ctx context.Context, param spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
        var fooLabelsWithName = "count_over_time({env=\"prod\"}[1h])"

        // get the chunks
        chunks, err := GetSpans(uint32(1000), fooLabelsWithName)
        operations := extractOperations(chunks)

        spans := make([]spanstore.Operation, 0, len(operations))
        for _, operation := range operations {
                if len(operation) > 0 {
                        spans = append(spans, spanstore.Operation{Name: operation})
                }
        }

        return spans, err
}

// GetTrace takes a traceID and returns a Trace associated with that traceID
func (r *Reader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
        log.Println("GetTrace executed")

        // will improvise this code later
        // traceID is in []model.TraceID{traceID}[0]
        var fooLabelsWithName = fmt.Sprintf("{env=\"prod\"} |= `trace_id_low=\"%d\"`", traceID.Low)
        log.Println("GetTrace Query: ", fooLabelsWithName)

        spans, err := GetSpansRange(r, fooLabelsWithName, time.Now().Add(time.Duration(-24) * time.Hour), time.Now(), uint32(100))
        chunks := spans.Result.Stream

        span := make([]*model.Span, 0, len(chunks)) 
        trace := make([]model.Trace_ProcessMapping, 0, len(chunks))
        for _, chunk := range chunks {
                var serviceName string
                var processId string
                var processTags map[string]interface{}

                // we decode the logfmt data in values
                // please refactor this decoder out to common code
                for _, value := range chunk.SValues {
                        d := logfmt.NewDecoder(strings.NewReader(string(value[1])))
                        for d.ScanRecord() {
                                for d.ScanKeyval() {
                                        if string(d.Key()) == "service_name" {
                                                serviceName = string(d.Value())
                                        }
                                        if string(d.Key()) == "process_id" {
                                                processId = string(d.Value())
                                        }
                                        if string(d.Key()) == "process_tags" {
                                                processTags = StrToMap(string(d.Value()))
                                        }
                                }
                        }
                        if d.Err() != nil {
                                log.Println("decoding logfmt error!", d.Err())
                        }
                        // end of decode

                        span = append(span, toModelSpanNew(value[1], chunk.SpanData))
                        trace = append(trace, model.Trace_ProcessMapping{
                                ProcessID: processId,
                                Process: model.Process{
                                        ServiceName: serviceName,
                                        Tags:        mapToModelKV(processTags),
                                },
                        })
                }
        } 

        return &model.Trace{Spans: span, ProcessMap: trace}, err
}

func buildTraceWhere(query *spanstore.TraceQueryParameters) (string, time.Time, time.Time) { 
        log.Println("buildTraceWhere executed")
        var builder string
        //log.Println("min time: %s", query.StartTimeMin)

        builder = "{"
        builder = builder + "env=\"prod\", "

        if len(query.ServiceName) > 0 {
                builder = builder + fmt.Sprintf("service_name = \"%s\", ", query.ServiceName)
        }
        if len(query.OperationName) > 0 {
                builder = builder + fmt.Sprintf("operation_name = \"%s\", ", query.OperationName)
        }

        /*
        if len(query.Tags) > 0 {
                for i, v := range query.Tags { 
                        builder = builder + fmt.Sprintf("tags =~ \".*%s:%s.*\", ", i, v)
                }
        }
        */

        // Remove last two characters (space and comma)
        builder = builder[:len(builder)-2]
        builder = builder + "}"

        // We are using logfmt to filter down the data
        if query.DurationMin > 0*time.Second || query.DurationMax > 0*time.Second {
                builder = builder + fmt.Sprintf(" | logfmt ")

        }

        /*
        if len(query.ServiceName) > 0 {
                builder = builder + fmt.Sprintf(" | service_name=\"%s\"", query.ServiceName)
        }
        if len(query.OperationName) > 0 {
                builder = builder + fmt.Sprintf(" | operation_name=\"%s\"", query.OperationName)
        }*/

        if len(query.Tags) > 0 {
                for i, v := range query.Tags {
                        builder = builder + fmt.Sprintf(" |~ \".*%s:%s.*\"", i, v)
                }
        }

        // filters
        // minimum duration in duration
        if query.DurationMin > 0*time.Second {
                builder = builder + fmt.Sprintf(" | latency > %s", time.Duration(query.DurationMin) / time.Nanosecond)
        }

        // max duration in duration
        if query.DurationMax > 0*time.Second {
                builder = builder + fmt.Sprintf(" | latency < %s", time.Duration(query.DurationMax) / time.Nanosecond)
        }

        // how many result of the traces do we want to show
        /* if query.NumTraces > 0 {
                builder = builder + fmt.Sprintf(" | limit = %d", query.NumTraces)
        } */

        // log our queries
        log.Println("builder: %s", builder)

        // here we include starttime min and max to pass to indexed timestamp
	return builder, query.StartTimeMin, query.StartTimeMax
}

// FindTraces retrieve traces that match the traceQuery
func (r *Reader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
       log.Println("FindTraces executed")

       builder, _, _ := buildTraceWhere(query)
       var fooLabelsWithName = builder

       m := make(map[string]bool)
       var traceIdsLow []string

       spans, err := GetSpansRange(r, fooLabelsWithName, query.StartTimeMin, query.StartTimeMax, uint32(query.NumTraces))
       chunks := spans.Result.Stream

       ret := make([]*model.Trace, 0, len(chunks))
       if err != nil {
               return ret, err
       }
       grouping := make(map[model.TraceID]*model.Trace)

       for _, chunk := range chunks {
               // we decode the logfmt data in values
               // please refactor this decoder out to common code
               for _, value := range chunk.SValues {

                       // query based on trace ID
                       d := logfmt.NewDecoder(strings.NewReader(value[1]))
                       for d.ScanRecord() {
                               for d.ScanKeyval() {
                                       if string(d.Key()) == "trace_id_low" {
                                               traceIdLow := string(d.Value()) 
                                                // make sure trace id is unique
                                               if !m[traceIdLow] {
                                                       traceIdsLow = append(traceIdsLow, traceIdLow)
                                                       m[traceIdLow] = true
                                               }
                                       }
                               }
                       }
                       if d.Err() != nil {
                               log.Println("decoding logfmt error!", d.Err())
                       }
                       // end of decode
               }       
       }

       // final query
       // now we get the real values
       for _, traceIDLow := range traceIdsLow {

               fooLabelsWithName = fmt.Sprintf("{env=\"prod\"} |= `trace_id_low=\"%s\"`", traceIDLow)
               relatedSpans, err := GetSpansRange(r, fooLabelsWithName, query.StartTimeMin, query.StartTimeMax, uint32(300))
               if err != nil {
                       log.Println("Unable to retrieve related Spans")
               }

               chunks := relatedSpans.Result.Stream

               for _, chunk := range chunks {
                       var serviceName string
                       var processId string
                       var processTags map[string]interface{}

                       for _, value := range chunk.SValues {
                               d := logfmt.NewDecoder(strings.NewReader(value[1]))
                               for d.ScanRecord() {
                                       for d.ScanKeyval() {
                                               if string(d.Key()) == "service_name" {
                                                       serviceName = string(d.Value())
                                               }
                                               if string(d.Key()) == "process_id" {
                                                       processId = string(d.Value())
                                               }
                                               if string(d.Key()) == "process_tags" {
                                                       processTags = StrToMap(string(d.Value()))
                                               }
                                      }
                               }
                               if d.Err() != nil {
                                      log.Println("decoding logfmt error!", d.Err())
                               }
                               // end of decode

                               modelSpan := toModelSpanNew(value[1], chunk.SpanData)
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
       }

       for _, trace := range grouping {
               ret = append(ret, trace)
       }

       return ret, err
}

// FindTraceIDs retrieve traceIDs that match the traceQuery
func (r *Reader) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) (ret []model.TraceID, err error) {
        builder, _, _ := buildTraceWhere(query)

        var fooLabelsWithName = builder

        spans, err := GetSpansRange(r, fooLabelsWithName, query.StartTimeMin, query.StartTimeMax, uint32(query.NumTraces))
        if err != nil {
                log.Println("Unable to get FindTraceIDs span!")
        }
        chunks := spans.Result.Stream

        var trace model.TraceID
        var traces []model.TraceID
        for _, chunk := range chunks {
                // we decode the logfmt data in values
                // please refactor this decoder out to common code
                for _, value := range chunk.SValues {

                        // query based on trace ID
                        d := logfmt.NewDecoder(strings.NewReader(value[1]))
                        for d.ScanRecord() {
                                for d.ScanKeyval() {
                                        if string(d.Key()) == "trace_id_low" {
                                                low, _ := strconv.ParseUint(string(d.Value()), 10, 64) 
                                                trace.Low = low
                                        }
                                        if string(d.Key()) == "trace_id_high" {
                                                high, _ := strconv.ParseUint(string(d.Value()), 10, 64)
                                                trace.High = high
                                        }
                                }
                        }
                        if d.Err() != nil {
                                log.Println("decoding logfmt error!", d.Err())
                        }
                        // end of decode
                        traces = append(traces, trace)

                }      
        }

        return traces, err
}

// GetDependencies returns all inter-service dependencies
func (r *Reader) GetDependencies(context context.Context, endTs time.Time, lookback time.Duration) (ret []model.DependencyLink, err error) {
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

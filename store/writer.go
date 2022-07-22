package loki

import (
	"io"
        "time"
        "context"
        "fmt"
        "log"
        "encoding/json"
        "net/http"
        "bytes"
        "io/ioutil"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"

        "github.com/weaveworks/common/user"

	pmodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"

	"github.com/grafana/loki/pkg/chunkenc"
	"github.com/grafana/loki/pkg/logql"
	"github.com/grafana/loki/pkg/logproto"
        "github.com/grafana/loki/pkg/storage"

        "github.com/grafana/loki/pkg/storage/chunk"
        "github.com/grafana/loki/pkg/ingester/client"

        hclog "github.com/hashicorp/go-hclog"
)

var _ spanstore.Writer = (*Writer)(nil)
var _ io.Closer = (*Writer)(nil)

//var fooLabelsWithName = "{foo=\"bar\", __name__=\"logs\"}"

var (
	ctx        = user.InjectOrgID(context.Background(), "fake")
)

// Writer handles all writes to object store for the Jaeger data model
type Writer struct {
       spanMeasurement     string
       spanMetaMeasurement string
       logMeasurement      string

       cfg                 Config
       store               storage.Store
       logger              hclog.Logger
}

type timeRange struct {
	from, to time.Time
}

type Stream struct {
       Env             string    `json:"env"`        
//       Id              string    `json:"id"`
//       TraceIDLow      string    `json:"trace_id_low"`
//       TraceIDHigh     string    `json:"trace_id_high"`
//       Flags           string    `json:"flags"`
//       Duration        string    `json:"duration"`
//       Tags            string    `json:"tags"`
//       ProcessId       string    `json:"process_id"`
//       ProcessTags     string    `json:"process_tags"`
//       Warnings        []string    `json:"warnings"`
       ServiceName     string    `json:"service_name"`
       OperationName   string    `json:"operation_name"`
//       StartTime       string    `json:"start_time"`
}

type Streams struct {
       Stream    Stream        `json:"stream"`    
       Values    [][]interface{}  `json:"values"`
}

type Data struct {
       Streams []Streams `json:"streams"`
}

func buildTestStreams(labels string, tr timeRange, line string) logproto.Stream {
        stream := logproto.Stream{
                Labels:  labels,
                Entries: []logproto.Entry{},
        }

        for from := tr.from; from.Before(tr.to); from = from.Add(time.Second) {
                stream.Entries = append(stream.Entries, logproto.Entry{
                        Timestamp: from,
                        Line:      line,
                })
        }

        return stream
}

func buildStreams(labels string, tr timeRange, line string) []logproto.Stream {
        stream := []logproto.Stream{
            {
                Labels:  labels,
                Entries: []logproto.Entry{},
            },
        }

        for from := tr.from; from.Before(tr.to); from = from.Add(time.Second) {
                stream[0].Entries = append(stream[0].Entries, logproto.Entry{
                        Timestamp: from,
                        Line:      line,
                })
        }

        return stream
}

func newChunk(stream logproto.Stream) chunk.Chunk {
        lbs, err := logql.ParseLabels(stream.Labels)
        if err != nil {
                panic(err)
        }
        if !lbs.Has(labels.MetricName) {
                builder := labels.NewBuilder(lbs)
                builder.Set(labels.MetricName, "logs")
                lbs = builder.Labels()
        }
        from, through := pmodel.TimeFromUnixNano(stream.Entries[0].Timestamp.UnixNano()), pmodel.TimeFromUnixNano(stream.Entries[0].Timestamp.UnixNano())
        chk := chunkenc.NewMemChunk(chunkenc.EncGZIP, chunkenc.UnorderedHeadBlockFmt, 256*1024, 0)
        for _, e := range stream.Entries {
                if e.Timestamp.UnixNano() < from.UnixNano() {
                        from = pmodel.TimeFromUnixNano(e.Timestamp.UnixNano())
                }
                if e.Timestamp.UnixNano() > through.UnixNano() {
                        through = pmodel.TimeFromUnixNano(e.Timestamp.UnixNano())
                }
                _ = chk.Append(&e)
        }
        chk.Close()
        c := chunk.NewChunk("fake", client.Fingerprint(lbs), lbs, chunkenc.NewFacade(chk, 0, 0), from, through)
        // force the checksum creation
        if err := c.Encode(); err != nil {
                panic(err)
        }
        return c
}

// NewWriter returns a Writer for object store
func NewWriter(cfg Config, store storage.Store, logger hclog.Logger) *Writer {
        return &Writer{
                cfg: cfg,
                store: store,
                logger: logger,
       }
}

// Close triggers a graceful shutdown
func (w *Writer) Close() error {
	return nil
}

// WriteSpan saves the span into object store
func (w *Writer) WriteSpan(context context.Context, span *model.Span) error {
        /* startTime := span.StartTime.Format(time.RFC3339)

        var spanLabelsWithName = fmt.Sprintf("{env=\"prod\", id=\"%d\", trace_id_low=\"%d\", trace_id_high=\"%d\", flags=\"%d\", duration=\"%d\", tags=\"%s\", process_id=\"%s\", process_tags=\"%s\", warnings=\"%s\", service_name=\"%s\", operation_name=\"%s\", start_time=\"%s\"}",
        span.SpanID,
        span.TraceID.Low,
        span.TraceID.High,
        span.Flags,
        span.Duration,
        mapModelKV(span.Tags),
        span.ProcessID,
        mapModelKV(span.Process.Tags),
        span.Warnings,
        span.Process.ServiceName,
        span.OperationName,
        startTime) 

        storeDate := time.Now()
        timeRanges := []timeRange{
                {
                        // chunk just for first store
                        storeDate,
                        storeDate.Add(span.Duration * time.Microsecond),
                },
        }

        plogline := fmt.Sprintf("level=info caller=jaeger component=chunks latency=\"%s\"", span.Duration)

        for _, tr := range timeRanges {
                req := logproto.PushRequest{
                        Streams: buildStreams(spanLabelsWithName, tr, plogline),
                }

                _, err := w.ingester.Push(ctx, &req)
                if err != nil  {
                        return err
                }
        } */

        startTime := span.StartTime.Format(time.RFC3339)
        plogline := fmt.Sprintf("level=info caller=jaeger component=chunks service_name=\"%s\" operation_name=\"%s\" start_time=\"%s\" latency=\"%s\" duration=\"%s\" span_id=\"%s\" trace_id_low=\"%d\" trace_id_high=\"%d\" flags=\"%d\" tags=\"%s\" process_id=\"%s\" process_tags=\"%s\" warnings=\"%s\"", span.Process.ServiceName, span.OperationName, startTime, span.Duration, span.Duration, span.SpanID, span.TraceID.Low, span.TraceID.High, span.Flags, mapModelKV(span.Tags), span.ProcessID, mapModelKV(span.Process.Tags), span.Warnings)

        var data Data
        /*
        data = Data{
            []Streams{
                {   
                    Stream{Env: "prod", Id: fmt.Sprintf("%d", span.SpanID), TraceIDLow: fmt.Sprintf("%d", span.TraceID.Low), TraceIDHigh: fmt.Sprintf("%d", span.TraceID.High), Flags: fmt.Sprintf("%d", span.Flags), Tags: mapModelKV(span.Tags), ProcessId: span.ProcessID, ProcessTags: mapModelKV(span.Process.Tags), Warnings: span.Warnings, ServiceName: span.Process.ServiceName, OperationName: span.OperationName},
                    [][]interface{}{{time.Now().UnixNano(), plogline}},
                },
            },
        } */

        // Avoid writing loki data, this will save us more storage
        if span.Process.ServiceName == "loki-all" {
                return nil
        }

        data = Data{
            []Streams{
                {
                    Stream{Env: "prod", ServiceName: span.Process.ServiceName, OperationName: span.OperationName},
                    [][]interface{}{{time.Now().UnixNano(), plogline}},
                },
            },
        }

        result, err := json.Marshal(data)
        if err != nil {
            return err
        }

        //log.Println("result %s", string(result))

	httpposturl := "http://localhost:4100/loki/api/v1/push"
	//fmt.Println("HTTP JSON POST URL:", httpposturl)

	request, error := http.NewRequest("POST", httpposturl, bytes.NewBuffer(result))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		return error
	}
	defer response.Body.Close()

        if response.Status != "204 No Content" {
                log.Println("Error! response Status:", response.Status)
                body, _ := ioutil.ReadAll(response.Body)
                log.Println("Error! response Body:", string(body))
        }
	//log.Println("response Status:", response.Status)
	//log.Println("response Headers:", response.Header)
	//body, _ := ioutil.ReadAll(response.Body)
	//log.Println("response Body:", string(body))

	return nil
}

func parseDate(in string) time.Time {
	t, err := time.Parse("2006-01-02", in)
	if err != nil {
		panic(err)
	}
	return t
}

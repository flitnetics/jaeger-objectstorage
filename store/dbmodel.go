package loki

import (
	"time"

	"github.com/jaegertracing/jaeger/model"
)

type Log struct {
	ID        uint64
	tableName struct{} `pg:"span_logs"`
	SpanID    model.SpanID
	Timestamp time.Time
	Fields    map[string]interface{}
}
type SpanRef struct {
	ID           uint64
	TraceIDLow   uint64
	TraceIDHigh  uint64
	SourceSpanID model.SpanID
	ChildSpanID  model.SpanID
	RefType      model.SpanRefType `sql:",use_zero"`
}
type Span struct {
	ID          model.SpanID
	TraceIDLow  uint64
	TraceIDHigh uint64
	Operation   *Operation
	OperationID uint
	Flags       model.Flags
	StartTime   time.Time
	Duration    time.Duration
	Tags        map[string]interface{}
	Service     *Service
	ServiceID   uint
	ProcessID   string
	ProcessTags map[string]interface{}
	Warnings    []string
	//References    []*SpanRef `pg:"fk:span_id"`
	//Logs          []*Log `pg:"fk:span_id"`
}
type Operation struct {
	ID            uint
	OperationName string `pg:",unique"`
}
type Service struct {
	ID          uint
	ServiceName string `pg:",unique"`
}

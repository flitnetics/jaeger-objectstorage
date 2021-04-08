package storage

import (
        "time"
       	"context"

	"github.com/jaegertracing/jaeger/model"
	"github.com/hashicorp/go-hclog"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

var _ spanstore.Reader = (*Reader)(nil)

// Reader can query for and load traces from InfluxDB v1.x.
type Reader struct {
	test            string
	logger hclog.Logger
}

// NewReader returns a new SpanReader for InfluxDB v1.x.
func NewReader(test string, logger hclog.Logger) *Reader {
	return &Reader{
                test:                test,
		logger:              logger,
	}
}

func (r *Reader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	r.logger.Warn("GetTrace called")

	return nil, nil
}

func (r *Reader) GetServices(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (r *Reader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	return nil, nil
}

func (r *Reader) GetOperations(ctx context.Context, param spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
	return nil, nil
}

func (r *Reader) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	return nil, nil
}

func (r *Reader) GetDependencies(ctx context.Context, ndTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return nil, nil
}

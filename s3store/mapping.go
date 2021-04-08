package s3store

import "github.com/jaegertracing/jaeger/model"

type whereBuilder struct {
	where  string
	params []interface{}
}

func (r *whereBuilder) andWhere(param interface{}, where string) {
	if len(r.where) > 0 {
		r.where += " AND "
	}
	r.where += where
	r.params = append(r.params, param)
}

func toModelSpan(span Span) *model.Span {
	return &model.Span{
		SpanID:        span.ID,
		TraceID:       model.TraceID{Low: span.TraceIDLow, High: span.TraceIDHigh},
		OperationName: span.Operation.OperationName,
		Flags:         span.Flags,
		StartTime:     span.StartTime,
		Duration:      span.Duration,
		Tags:          mapToModelKV(span.Tags),
		ProcessID:     span.ProcessID,
		Process: &model.Process{
			ServiceName: span.Service.ServiceName,
			Tags:        mapToModelKV(span.ProcessTags),
		},
		Warnings:   span.Warnings,
		References: make([]model.SpanRef, 0),
		Logs:       make([]model.Log, 0),
	}
}

func mapToModelKV(input map[string]interface{}) []model.KeyValue {
	ret := make([]model.KeyValue, 0, len(input))
	var kv model.KeyValue
	for k, v := range input {
		if vStr, ok := v.(string); ok {
			kv = model.KeyValue{
				Key:   k,
				VType: model.ValueType_STRING,
				VStr:  vStr,
			}
			ret = append(ret, kv)
		} else if vBytes, ok := v.([]byte); ok {
			kv = model.KeyValue{
				Key:     k,
				VType:   model.ValueType_BINARY,
				VBinary: vBytes,
			}
			ret = append(ret, kv)
		} else if vBool, ok := v.(bool); ok {
			kv = model.KeyValue{
				Key:   k,
				VType: model.ValueType_BOOL,
				VBool: vBool,
			}
			ret = append(ret, kv)
		} else if vInt64, ok := v.(int64); ok {
			kv = model.KeyValue{
				Key:    k,
				VType:  model.ValueType_INT64,
				VInt64: vInt64,
			}
			ret = append(ret, kv)
		} else if vFloat64, ok := v.(float64); ok {
			kv = model.KeyValue{
				Key:      k,
				VType:    model.ValueType_FLOAT64,
				VFloat64: vFloat64,
			}
			ret = append(ret, kv)
		}
	}
	return ret
}

func mapModelKV(input []model.KeyValue) map[string]interface{} {
	ret := make(map[string]interface{})
	var value interface{}
	for _, kv := range input {
		value = nil
		if kv.VType == model.ValueType_STRING {
			value = kv.VStr
		} else if kv.VType == model.ValueType_BOOL {
			value = kv.VBool
		} else if kv.VType == model.ValueType_INT64 {
			value = kv.VInt64
		} else if kv.VType == model.ValueType_FLOAT64 {
			value = kv.VFloat64
		} else if kv.VType == model.ValueType_BINARY {
			value = kv.VBinary
		}
		ret[kv.Key] = value
	}
	return ret
}

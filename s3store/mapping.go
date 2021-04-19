package s3store

import (
        "github.com/jaegertracing/jaeger/model"
        "github.com/cortexproject/cortex/pkg/chunk"
        "strconv"
        "time"
        "strings"
)

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

func StrToMap(in string) map[string]interface{} {
    res := make(map[string]interface{})
    array := strings.Split(in, " ")
    temp := make([]string, 2)
    for _, val := range array {
        temp = strings.Split(string(val), ":")
        res[temp[0]] = temp[1]
    }
    return res
}

func toModelSpan(chunk chunk.Chunk) *model.Span {
        var id model.SpanID
        if (chunk.Metric[2].Name == "id") {
                conv, _ := strconv.ParseUint(chunk.Metric[2].Value, 10, 64)
                id = model.NewSpanID(conv)
        }

        var trace_id_low uint64
        if (chunk.Metric[2].Name == "trace_id_low") {
                trace_id_low, _ = strconv.ParseUint(chunk.Metric[2].Value, 10, 64)
        }

        var trace_id_high uint64
        if (chunk.Metric[2].Name == "trace_id_high") {
                trace_id_high, _ = strconv.ParseUint(chunk.Metric[2].Value, 10, 64)
        }

        var operation_name string 
        if (chunk.Metric[2].Name == "operation_name") {
                operation_name = chunk.Metric[2].Value
        }

        var flags model.Flags
        if (chunk.Metric[2].Name == "flags") {
                conv, _ := strconv.ParseUint(chunk.Metric[2].Value, 10, 64)
                flags = model.Flags(conv)
        }
         
        var duration time.Duration
        if (chunk.Metric[2].Name == "duration") {
                conv, _ := strconv.ParseUint(chunk.Metric[2].Value, 10, 64)
                duration = time.Duration(conv)
        }

        var tags map[string]interface{}
        if (chunk.Metric[2].Name == "tags") {
                tags = StrToMap(chunk.Metric[2].Value)
        }

        var process_id string
        if (chunk.Metric[2].Name == "process_id") {
                process_id = chunk.Metric[2].Value
        }

        var service_name string
        if (chunk.Metric[2].Name == "service_name") {
                service_name = chunk.Metric[2].Value
        }

        var process_tags map[string]interface{}
        if (chunk.Metric[2].Name == "process_tags") {
                process_tags = StrToMap(chunk.Metric[2].Value)
        }

        /* var warnings string
        if (chunk.Metric[2].Name == "warnings") {
                warnings = chunk.Metric[2].Value
        } */

        var start_time time.Time
        if (chunk.Metric[2].Name == "start_time") {
                layout := "2006-01-02T15:04:05.000Z"
                start_time, _ = time.Parse(layout, chunk.Metric[2].Value)
        }

	return &model.Span{
		SpanID:        id,
		TraceID:       model.TraceID{Low: trace_id_low, High: trace_id_high},
		OperationName: operation_name,
		Flags:         flags,
		StartTime:     start_time,
		Duration:      duration,
		Tags:          mapToModelKV(tags),
		ProcessID:     process_id,
		Process: &model.Process{
			ServiceName: service_name,
			Tags:        mapToModelKV(process_tags),
		},
		//Warnings:   warnings,
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

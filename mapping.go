package store

import (
        "github.com/jaegertracing/jaeger/model"
        "strconv"
)

func mapModelKV(input []model.KeyValue) string {
        ret := ""
	var value interface{}
	for _, kv := range input {
		value = nil
		if kv.VType == model.ValueType_STRING {
			value = kv.VStr
		} else if kv.VType == model.ValueType_BOOL {
			value = strconv.FormatBool(kv.VBool)
		} else if kv.VType == model.ValueType_INT64 {
			value = strconv.FormatInt(int64(kv.VInt64), 10)
		} else if kv.VType == model.ValueType_FLOAT64 {
			value = strconv.FormatFloat(kv.VFloat64, 'E', -1, 64)
		} else if kv.VType == model.ValueType_BINARY {
			value = kv.VBinary
		}
		ret = ret + fmt.Sprintf("%s:%s", kv.Key, value) + " "
	}
	return ret
}

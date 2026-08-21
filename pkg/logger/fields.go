package logger

import "time"

// Field 表示一个结构化日志字段。
type Field struct {
	Key   string
	Value interface{}
}

// String 构造字符串字段。
func String(key, value string) Field { return Field{Key: key, Value: value} }

// Int 构造整型字段。
func Int(key string, value int) Field { return Field{Key: key, Value: value} }

// Int64 构造 int64 字段。
func Int64(key string, value int64) Field { return Field{Key: key, Value: value} }

// Float64 构造 float64 字段。
func Float64(key string, value float64) Field { return Field{Key: key, Value: value} }

// Bool 构造布尔字段。
func Bool(key string, value bool) Field { return Field{Key: key, Value: value} }

// Duration 构造时长字段，值为毫秒数。
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.Milliseconds()}
}

// Err 构造错误字段，键固定为 "error"。
func Err(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: ""}
	}
	return Field{Key: "error", Value: err.Error()}
}

// ToKV 将字段切片展开为 key, value 交替的切片，供日志方法使用。
func ToKV(fields []Field) []interface{} {
	kv := make([]interface{}, 0, len(fields)*2)
	for _, f := range fields {
		kv = append(kv, f.Key, f.Value)
	}
	return kv
}

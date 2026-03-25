package convert

import (
	"fmt"
	"time"
)

func CentsToFloat(cents int64) float64 {
	return float64(cents) / 100
}

func FloatToCents(amount float64) int64 {
	return int64(amount * 100)
}

func ParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func ParseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := ParseTime(s)
	return &t
}

func UnixToTime(ts int64) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

func UnixToTimePtr(ts int64) *time.Time {
	if ts == 0 {
		return nil
	}
	t := time.Unix(ts, 0)
	return &t
}

func TimeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func StringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func Int64FromMap(m map[string]interface{}, key string) int64 {
	switch val := m[key].(type) {
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case int64:
		return val
	}
	return 0
}

func Float64FromMap(m map[string]interface{}, key string) float64 {
	switch val := m[key].(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	}
	return 0
}

func BoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return false
}

func MetadataToStringMap(m map[string]interface{}) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range m {
		if str, ok := v.(string); ok {
			result[k] = str
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

func InterfaceToStringMap(m interface{}) map[string]string {
	if m == nil {
		return nil
	}
	if metadataMap, ok := m.(map[string]interface{}); ok {
		return MetadataToStringMap(metadataMap)
	}
	return nil
}

func StringMapToMetadata(m map[string]string) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

func MapFromInterface(m interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	if result, ok := m.(map[string]interface{}); ok {
		return result
	}
	return nil
}

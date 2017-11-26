package util

import "encoding/json"

func StructToBoolMap(data interface{}) map[string]bool {
	result := make(map[string]bool)

	b, _ := json.Marshal(data)
	json.Unmarshal(b, &result)

	return result
}

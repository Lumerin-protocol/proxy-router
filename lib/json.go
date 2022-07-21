package lib

import "encoding/json"

func NormalizeJson(msg []byte) ([]byte, error) {
	var a map[string]interface{}
	json.Unmarshal(msg, &a)
	return json.Marshal(a)
}

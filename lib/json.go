package lib

import "encoding/json"

func NormalizeJson(msg []byte) ([]byte, error) {
	var a map[string]interface{}
	err := json.Unmarshal(msg, &a)
	if err != nil {
		return nil, err
	}
	return json.Marshal(a)
}

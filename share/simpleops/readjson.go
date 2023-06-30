package simpleops

import (
	"encoding/json"
	"os"
)

func ReadJSONFileIntoStruct[T any](path string) (T, error) {
	var tmp T
	data, err := os.ReadFile(path)
	if err != nil {
		return tmp, err
	}

	err = json.Unmarshal(data, &tmp)

	return tmp, err
}

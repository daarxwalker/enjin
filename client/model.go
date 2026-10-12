package client

import "encoding/json"

type valuesDoc struct {
	Values    map[string]any   `json:"values"`
	UnixMilli map[string]int64 `json:"unixMilli"`
}

type storedValuesDoc struct {
	Values    map[string]json.RawMessage `json:"values"`
	UnixMilli map[string]int64           `json:"unixMilli"`
}

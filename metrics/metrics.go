package metrics

import (
	"encoding/json"

	benchmarks "github.com/rotationalio/ensign-benchmarks"
)

type Metrics map[string]interface{}

var _ benchmarks.Metrics = make(Metrics)

func (m Metrics) Measurements() ([]string, error) {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys, nil
}

func (m Metrics) Measurement(name string) interface{} {
	return m[name]
}

func (m Metrics) MarshalJSON() ([]byte, error) {
	v := map[string]interface{}(m)
	return json.Marshal(v)
}

package bq

import (
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
)

func TestRowToMetric(t *testing.T) {
	tests := []struct {
		name   string
		row    map[string]bigquery.Value
		metric Metric
	}{
		{
			name: "Extract labels",
			row: map[string]bigquery.Value{
				"machine": "mlab1.foo01.measurement-lab.org",
				"value":   1.0,
			},
			metric: Metric{
				labelKeys:   []string{"machine"},
				labelValues: []string{"mlab1.foo01.measurement-lab.org"},
				values:      map[string]float64{"": 1.0},
			},
		},
		{
			name: "No labels",
			row: map[string]bigquery.Value{
				"value": 1.1,
			},
			metric: Metric{
				labelKeys:   nil,
				labelValues: nil,
				values:      map[string]float64{"": 1.1},
			},
		},
		{
			name: "Integer value",
			row: map[string]bigquery.Value{
				"value": int64(10),
			},
			metric: Metric{
				labelKeys:   nil,
				labelValues: nil,
				values:      map[string]float64{"": 10},
			},
		},
		{
			name: "Multiple values",
			row: map[string]bigquery.Value{
				"value_foo": int64(10),
				"value_bar": int64(20),
			},
			metric: Metric{
				labelKeys:   nil,
				labelValues: nil,
				values:      map[string]float64{"_foo": 10, "_bar": 20},
			},
		},
		{
			name: "Bad label values are converted to strings",
			row: map[string]bigquery.Value{
				"name":  3.0, // should be a string.
				"value": 2.1,
			},
			metric: Metric{
				labelKeys:   []string{"name"},
				labelValues: []string{"invalid string"}, // converted to a string.
				values:      map[string]float64{"": 2.1},
			},
		},
		{
			name: "Convert time.Time type into string",
			row: map[string]bigquery.Value{
				"timestamp": time.Date(2019, time.September, 26, 0, 0, 0, 0, time.UTC), //
				"value":     1.0,
			},
			metric: Metric{
				labelKeys:   []string{"timestamp"},
				labelValues: []string{"2019-09-26 00:00:00 +0000 UTC"}, // converted to a string.
				values:      map[string]float64{"": 1.0},
			},
		},
	}

	for _, test := range tests {
		m := rowToMetric(test.row)
		if !reflect.DeepEqual(m, test.metric) {
			t.Errorf("Failed to convert row to metric. want %#v; got %#v", test.metric, m)
		}
	}
}

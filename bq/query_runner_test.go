package bq

import (
	"reflect"
	"testing"

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
				"label_machine": "mlab1.foo01.measurement-lab.org",
				"value":         1.0,
			},
			metric: Metric{
				labels: []string{"machine"},
				values: []string{"mlab1.foo01.measurement-lab.org"},
				value:  1.0,
			},
		},
		{
			name: "No labels",
			row: map[string]bigquery.Value{
				"value": 1.1,
			},
			metric: Metric{
				labels: nil,
				values: nil,
				value:  1.1,
			},
		},
		{
			name: "Bad label values are converted to strings",
			row: map[string]bigquery.Value{
				"label_name": 3.0, // should be a string.
				"value":      2.1,
			},
			metric: Metric{
				labels: []string{"name"},
				values: []string{"3.00"}, // converted to a string.
				value:  2.1,
			},
		},
	}

	for _, test := range tests {
		m := rowToMetric(test.row)
		if !reflect.DeepEqual(m, test.metric) {
			t.Error("Failed to convert row to metric. want %#v; got %#v", test.metric, m)
		}
	}
}

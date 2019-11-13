package query

import (
	"reflect"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-lab/prometheus-bigquery-exporter/sql"
)

func TestRowToMetric(t *testing.T) {
	tests := []struct {
		name   string
		row    map[string]bigquery.Value
		metric sql.Metric
	}{
		{
			name: "Extract labels",
			row: map[string]bigquery.Value{
				"machine": "mlab1.foo01.measurement-lab.org",
				"value":   1.0,
			},
			metric: sql.Metric{
				LabelKeys:   []string{"machine"},
				LabelValues: []string{"mlab1.foo01.measurement-lab.org"},
				Values:      map[string]float64{"": 1.0},
			},
		},
		{
			name: "No labels",
			row: map[string]bigquery.Value{
				"value": 1.1,
			},
			metric: sql.Metric{
				LabelKeys:   nil,
				LabelValues: nil,
				Values:      map[string]float64{"": 1.1},
			},
		},
		{
			name: "Integer value",
			row: map[string]bigquery.Value{
				"value": int64(10),
			},
			metric: sql.Metric{
				LabelKeys:   nil,
				LabelValues: nil,
				Values:      map[string]float64{"": 10},
			},
		},
		{
			name: "Multiple values",
			row: map[string]bigquery.Value{
				"value_foo": int64(10),
				"value_bar": int64(20),
			},
			metric: sql.Metric{
				LabelKeys:   nil,
				LabelValues: nil,
				Values:      map[string]float64{"_foo": 10, "_bar": 20},
			},
		},
		{
			name: "Bad label values are converted to strings",
			row: map[string]bigquery.Value{
				"name":  3.0, // should be a string.
				"value": 2.1,
			},
			metric: sql.Metric{
				LabelKeys:   []string{"name"},
				LabelValues: []string{"invalid string"}, // converted to a string.
				Values:      map[string]float64{"": 2.1},
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

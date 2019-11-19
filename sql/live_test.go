package sql_test

import (
	"context"
	"reflect"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-lab/prometheus-bigquery-exporter/query"
	"github.com/m-lab/prometheus-bigquery-exporter/sql"
)

// TestLiveQuery uses a real BigQuery client to connect and run actual queries.
// Requires auth credentials.
// Note: disable this test in travis by specifying "go test -short [...]"
func TestLiveQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live query tests.")
	}
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "measurement-lab")
	if err != nil {
		t.Fatal(err)
	}
	qr := query.NewBQRunner(client)
	tests := []struct {
		name    string
		query   string
		metrics []sql.Metric
	}{
		{
			name:  "Single row, single value",
			query: "SELECT 1 as value",
			metrics: []sql.Metric{
				sql.NewMetric(nil, nil, map[string]float64{"": 1.0}),
			},
		},
		{
			name:  "Single row, single value with label",
			query: "SELECT 'foo' as key, 2 as value",
			metrics: []sql.Metric{
				sql.NewMetric([]string{"key"}, []string{"foo"}, map[string]float64{"": 2.0}),
			},
		},
		{
			name: "Multiple rows, single value with labels",
			query: `#standardSQL
			        SELECT key, value
					FROM (SELECT "foo" AS key, 1 AS value UNION ALL
					      SELECT "bar" AS key, 2 AS value);`,
			metrics: []sql.Metric{
				sql.NewMetric([]string{"key"}, []string{"foo"}, map[string]float64{"": 1.0}),
				sql.NewMetric([]string{"key"}, []string{"bar"}, map[string]float64{"": 2.0}),
			},
		},
		{
			name: "Multiple rows, multiple values with labels",
			query: `#standardSQL
			        SELECT key, value_foo, value_bar
					FROM (SELECT "foo" AS key, 1 AS value_foo, 3 as value_bar UNION ALL
					      SELECT "bar" AS key, 2 AS value_foo, 4 as value_bar);`,
			metrics: []sql.Metric{
				sql.NewMetric([]string{"key"}, []string{"foo"}, map[string]float64{"_foo": 1.0, "_bar": 3.0}),
				sql.NewMetric([]string{"key"}, []string{"bar"}, map[string]float64{"_foo": 2.0, "_bar": 4.0}),
			},
		},
	}
	for _, test := range tests {
		t.Logf("Live query test: %s", test.name)
		metrics, err := qr.Query(test.query)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(metrics, test.metrics) {
			t.Errorf("Metrics do not match:\nwant %#v;\n got %#v", test.metrics, metrics)
		}
	}
}

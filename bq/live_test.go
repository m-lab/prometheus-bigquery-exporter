package bq_test

import (
	"context"
	"reflect"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-lab/prometheus-bigquery-exporter/bq"
)

// TestLiveQuery uses a real BigQuery client to connect and run actual queries.
// Requires auth credentials.
// Note: disable this test in travis by specifying "go test -short [...]"
func TestLiveQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping live query tests.")
	}
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "mlab-sandbox")
	if err != nil {
		t.Fatal(err)
	}
	qr := bq.NewQueryRunner(client)
	tests := []struct {
		name    string
		query   string
		metrics []bq.Metric
	}{
		{
			name:  "Single value",
			query: "SELECT 1 as value",
			metrics: []bq.Metric{
				bq.NewMetric(nil, nil, 1.0),
			},
		},
		{
			name:  "Single value with label",
			query: "SELECT 'foo' as key, 2 as value",
			metrics: []bq.Metric{
				bq.NewMetric([]string{"key"}, []string{"foo"}, 2.0),
			},
		},
		{
			name: "Multiple values with labels",
			query: `#standardSQL
			        SELECT key, value
					FROM (SELECT "foo" AS key, 1 AS value UNION ALL
					      SELECT "bar" AS key, 2 AS value);`,
			metrics: []bq.Metric{
				bq.NewMetric([]string{"key"}, []string{"foo"}, 1.0),
				bq.NewMetric([]string{"key"}, []string{"bar"}, 2.0),
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

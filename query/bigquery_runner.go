// Package query defines the QueryRunner interface for running bigquery queries
// and converting the results into a set of key/value labels and a single
// float64 value suitable for converting into a prometheus.Metric.
package query

import (
	"context"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/m-lab/prometheus-bigquery-exporter/sql"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// BQRunner is a concerete implementation of QueryRunner for BigQuery.
type BQRunner struct {
	client *bigquery.Client
}

// NewBQRunner creates a new QueryRunner instance.
func NewBQRunner(client *bigquery.Client) *BQRunner {
	return &BQRunner{client}
}

// Query executes the given query. Query only supports standard SQL. The
// query must define a column named "value" for the value, and may define
// additional columns, all of which are used as metric labels.
func (qr *BQRunner) Query(query string) ([]sql.Metric, error) {
	metrics := []sql.Metric{}

	q := qr.client.Query(query)
	// TODO: add context timeout.
	it, err := q.Read(context.Background())
	if err != nil {
		log.Print(err)
		return nil, err
	}

	for {
		var row map[string]bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("%#v %d", err, len(metrics))
			return nil, err
		}
		metrics = append(metrics, rowToMetric(row))
	}
	return metrics, nil
}

// valToFloat extracts a float from the bigquery.Value irrespective of the
// underlying type. If the type is not int64, float64, then valToFloat returns
// zero.
func valToFloat(v bigquery.Value) float64 {
	switch v.(type) {
	case int64:
		return (float64)(v.(int64))
	case float64:
		return v.(float64)
	default:
		return math.NaN()
	}
}

// valToString coerces the bigquery.Value into a string. If the underlying type
// is not a string, we treat it like an int64 or float64. If the underlying is
// none of these types, valToString returns "invalid string".
func valToString(v bigquery.Value) string {
	var s string
	switch v.(type) {
	case string:
		s = v.(string)
	default:
		s = "invalid string"
	}
	return s
}

// rowToMetric converts a bigquery result row to a bq.Metric
func rowToMetric(row map[string]bigquery.Value) sql.Metric {
	values := make(map[string]float64, 1)
	var labelKeys []string
	var labelValues []string

	// Note that `range` does not guarantee map key order. So, we extract label
	// names, sort them, and then extract values.
	for k, v := range row {
		if strings.HasPrefix(k, "value") {
			// Get the value suffix used to augment the metric name. If k is
			// "value", then the default name will just be the empty string.
			values[k[5:]] = valToFloat(v)
		} else {
			labelKeys = append(labelKeys, k)
		}
	}
	sort.Strings(labelKeys)

	for i := range labelKeys {
		labelValues = append(labelValues, valToString(row[labelKeys[i]]))
	}
	return sql.NewMetric(labelKeys, labelValues, values)
}

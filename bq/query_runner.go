// query_runner defines the QueryRunner interface for running bigquery queries
// and converting the results into a set of key/value labels and a single
// float64 value suitable for converting into a prometheus.Metric.
package bq

import (
	"context"
	"log"
	"math"
	"sort"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// QueryRunner defines the interface used to run a query and return an array of metrics.
type QueryRunner interface {
	Query(q string) ([]Metric, error)
}

// Metric holds raw data from query results needed to create a prometheus.Metric.
type Metric struct {
	labels []string
	values []string
	value  float64
}

// queryRunnerImpl is a concerete implementation of QueryRunner for BigQuery.
type queryRunnerImpl struct {
	client *bigquery.Client
}

// NewQueryRunner creates a new QueryRunner instance.
func NewQueryRunner(client *bigquery.Client) QueryRunner {
	return &queryRunnerImpl{client}
}

// Query executes the given query. Currently only Legacy SQL is supported. The
// query must define a column named "value" for the value, and may define label
// columns that use the prefix "label_".
func (qr *queryRunnerImpl) Query(query string) ([]Metric, error) {
	metrics := []Metric{}

	q := qr.client.Query(query)

	// TODO: only use Standard SQL.
	q.QueryConfig.UseLegacySQL = true

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
// none of these types, valToString returns "0".
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
func rowToMetric(row map[string]bigquery.Value) Metric {
	m := Metric{}

	// Note that `range` does not guarantee map key order. So, we extract label
	// names, sort them, and then extract values.
	for k, v := range row {
		if k == "value" {
			m.value = valToFloat(v)
		} else {
			m.labels = append(m.labels, k)
		}
	}
	sort.Strings(m.labels)

	for i := range m.labels {
		m.values = append(m.values, valToString(row[m.labels[i]]))
	}
	return m
}

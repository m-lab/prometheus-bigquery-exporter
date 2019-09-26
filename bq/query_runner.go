// query_runner defines the QueryRunner interface for running bigquery queries
// and converting the results into a set of key/value labels and a single
// float64 value suitable for converting into a prometheus.Metric.
package bq

import (
	"context"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// QueryRunner defines the interface used to run a query and return an array of metrics.
type QueryRunner interface {
	Query(q string) ([]Metric, error)
}

// Metric holds raw data from query results needed to create a prometheus.Metric.
type Metric struct {
	labelKeys   []string
	labelValues []string
	values      map[string]float64
}

// NewMetric creates a Metric with given values.
func NewMetric(labelKeys []string, labelValues []string, values map[string]float64) Metric {
	return Metric{labelKeys, labelValues, values}
}

// queryRunnerImpl is a concerete implementation of QueryRunner for BigQuery.
type queryRunnerImpl struct {
	client *bigquery.Client
}

// NewQueryRunner creates a new QueryRunner instance.
func NewQueryRunner(client *bigquery.Client) QueryRunner {
	return &queryRunnerImpl{client}
}

// Query executes the given query. Query only supports standard SQL. The
// query must define a column named "value" for the value, and may define
// additional columns, all of which are used as metric labels.
func (qr *queryRunnerImpl) Query(query string) ([]Metric, error) {
	metrics := []Metric{}

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
// is not a string, we check against other special types such as time.Time and
// try to convert to a string. If the underlying is none of these types, valToString
// returns "invalid string".
func valToString(v bigquery.Value) string {
	var s string
	switch v.(type) {
	case string:
		s = v.(string)
	case time.Time:
		s = v.(time.Time).String()
	default:
		s = "invalid string"
	}
	return s
}

// rowToMetric converts a bigquery result row to a bq.Metric
func rowToMetric(row map[string]bigquery.Value) Metric {
	m := Metric{}
	m.values = make(map[string]float64, 1)

	// Note that `range` does not guarantee map key order. So, we extract label
	// names, sort them, and then extract values.
	for k, v := range row {
		if strings.HasPrefix(k, "value") {
			// Get the value suffix used to augment the metric name. If k is
			// "value", then the default name will just be the empty string.
			m.values[k[5:]] = valToFloat(v)
		} else {
			m.labelKeys = append(m.labelKeys, k)
		}
	}
	sort.Strings(m.labelKeys)

	for i := range m.labelKeys {
		m.labelValues = append(m.labelValues, valToString(row[m.labelKeys[i]]))
	}
	return m
}

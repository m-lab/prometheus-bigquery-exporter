// Package query defines the QueryRunner interface for running bigquery queries
// and converting the results into a set of key/value labels and a single
// float64 value suitable for converting into a prometheus.Metric.
package query

import (
	"context"
	"errors"
	"log"
	"math"
	"sort"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/m-lab/prometheus-bigquery-exporter/sql"
	"google.golang.org/api/iterator"
)

type bigQueryImpl struct {
	bigquery.Client
}

func (b *bigQueryImpl) Query(query string, visit func(row map[string]bigquery.Value) error) error {
	ctx := context.Background()

	q := b.Client.Query(query)

	job, err := q.Run(ctx)
	if err != nil {
		return err
	}

	js, err := job.Wait(ctx)
	if err != nil {
		return err
	}

	var it *bigquery.RowIterator

	if js.Statistics.NumChildJobs > 0 {
		it, err = queryChildren(ctx, job)
	} else {
		it, err = job.Read(ctx)
	}

	if err != nil {
		return err
	}
	var row map[string]bigquery.Value
	for err = it.Next(&row); err == nil; err = it.Next(&row) {
		err2 := visit(row)
		if err2 != nil {
			return err2
		}
	}
	if err != iterator.Done {
		return err
	}
	return nil
}

func queryChildren(ctx context.Context, job *bigquery.Job) (*bigquery.RowIterator, error) {
	var childJobs []*bigquery.Job

	it := job.Children(ctx)
	for {
		job, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		childJobs = append(childJobs, job)
	}
	if len(childJobs) == 0 {
		return nil, errors.New("could not find any child jobs despite having them")
	}

	return childJobs[0].Read(ctx)
}

// BQRunner is a concrete implementation of QueryRunner for BigQuery.
type BQRunner struct {
	runner runner
}

// runner interface allows unit testing of the Query function.
type runner interface {
	Query(q string, visit func(row map[string]bigquery.Value) error) error
}

// NewBQRunner creates a new QueryRunner instance.
func NewBQRunner(client *bigquery.Client) *BQRunner {
	return &BQRunner{
		runner: &bigQueryImpl{
			Client: *client,
		},
	}
}

// Query executes the given query. Query only supports standard SQL. The
// query must define a column prefixed with "value" for the value(s), and may define
// additional columns, all of which are used as metric labels.
func (qr *BQRunner) Query(query string) ([]sql.Metric, error) {
	metrics := []sql.Metric{}
	err := qr.runner.Query(query, func(row map[string]bigquery.Value) error {
		metrics = append(metrics, rowToMetric(row))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

// valToFloat extracts a float from the bigquery.Value irrespective of the
// underlying type. If the type is not int64, float64, then valToFloat returns
// zero.
func valToFloat(v bigquery.Value) float64 {
	if v == nil {
		return 0
	}
	switch v.(type) {
	case int64:
		return (float64)(v.(int64))
	case float64:
		return v.(float64)
	default:
		log.Printf("Unrecognized value format: %T", v)
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

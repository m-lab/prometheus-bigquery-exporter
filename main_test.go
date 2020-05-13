// bigquery_exporter runs structured bigquery SQL and converts the results into
// prometheus metrics. bigquery_exporter can process multiple queries.
// Because BigQuery queries can have long run times and high cost, Query results
// are cached and updated every refresh interval, not on every scrape of
// prometheus metrics.
package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/m-lab/go/rtx"
	"github.com/m-lab/prometheus-bigquery-exporter/sql"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

type fakeRunner struct {
	updated int
}

func (f *fakeRunner) Query(query string) ([]sql.Metric, error) {
	r := []sql.Metric{
		{
			LabelKeys:   []string{"key"},
			LabelValues: []string{"value"},
			Values: map[string]float64{
				"okay": 1.23,
			},
		},
	}
	f.updated++
	if f.updated > 1 {
		// Simulate an error after one successful query.
		return nil, fmt.Errorf("Fake failure for testing")
	}
	return r, nil
}

func Test_main(t *testing.T) {
	tmp, err := ioutil.TempFile("", "empty_query_*")
	rtx.Must(err, "Failed to create temp file for main test.")
	defer os.Remove(tmp.Name())

	// Provide coverage of the original newRunner definition.
	newRunner(nil)

	// Create a fake runner for the test.
	f := &fakeRunner{}
	newRunner = func(*bigquery.Client) sql.QueryRunner {
		return f
	}

	// Set the refresh period to a very small delay.
	*refresh = time.Second
	gaugeSources.Set(tmp.Name())

	// Reset mainCtx to timeout after a second.
	mainCtx, mainCancel = context.WithTimeout(mainCtx, time.Second)
	defer mainCancel()

	main()

	// Verify that the fakeRunner was called twice.
	if f.updated != 2 {
		t.Errorf("main() failed to update; got %d, want 2", f.updated)
	}
}

// bq implements the prometheus.Collector interface for bigquery.
package bq

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	runner     QueryRunner
	metricName string
	query      string

	valType prometheus.ValueType
	desc    *prometheus.Desc

	metrics []Metric
	mux     sync.Mutex
}

// NewCollector creates a new BigQuery Collector instance.
func NewCollector(runner QueryRunner, valType prometheus.ValueType, metricName, query string) *Collector {
	return &Collector{
		runner:     runner,
		metricName: metricName,
		query:      query,
		valType:    valType,
		desc:       nil,
		metrics:    nil,
		mux:        sync.Mutex{},
	}
}

// Describe satisfies the prometheus.Collector interface. Describe is called
// immediately after registering the collector.
func (col *Collector) Describe(ch chan<- *prometheus.Desc) {
	if col.desc == nil {
		// TODO: collect metrics for query exec time.
		col.Update()
		col.setDesc()
	}
	// NOTE: if Update returns no metrics, this will fail.
	ch <- col.desc
}

// Collect satisfies the prometheus.Collector interface. Collect reports values
// from cached metrics.
func (col *Collector) Collect(ch chan<- prometheus.Metric) {
	col.mux.Lock()
	// Get reference to current metrics slice to allow Update to run concurrently.
	metrics := col.metrics
	col.mux.Unlock()

	for i := range col.metrics {
		ch <- prometheus.MustNewConstMetric(
			col.desc, col.valType, metrics[i].value, metrics[i].values...)
	}
}

// String satisfies the Stringer interface. String returns the metric name.
func (col *Collector) String() string {
	return col.metricName
}

// Update runs the collector query and atomically updates the cached metrics.
// Update is called automaticlly after the collector is registered.
func (col *Collector) Update() error {
	metrics, err := col.runner.Query(col.query)
	if err != nil {
		return err
	}
	// Swap the cached metrics.
	col.mux.Lock()
	defer col.mux.Unlock()
	// Replace slice reference with new value returned from Query. References
	// to the previous value of col.metrics are not affected.
	col.metrics = metrics
	return nil
}

func (col *Collector) setDesc() {
	// The query may return no results.
	if len(col.metrics) > 0 {
		// TODO: allow passing meaningful help text.
		col.desc = prometheus.NewDesc(col.metricName, "help text", col.metrics[0].labels, nil)
	} else {
		// TODO: this is a problem.
		return
	}
}

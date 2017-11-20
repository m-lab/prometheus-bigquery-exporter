// bq implements the prometheus.Collector interface for bigquery.
package bq

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	// runner must be a QueryRunner instance for collecting metrics.
	runner QueryRunner
	// metricName is the base name for prometheus metrics created for this query.
	metricName string
	// query contains the standardSQL query.
	query string

	// valType defines whether the metric is a Gauge or Counter type.
	valType prometheus.ValueType
	// descs maps metric suffixes to the prometheus description. These descriptions
	// are generated once and must be stable over time.
	descs map[string]*prometheus.Desc

	// metrics caches the last set of collected results from a query.
	metrics []Metric
	// mux locks access to types above.
	mux sync.Mutex
}

// NewCollector creates a new BigQuery Collector instance.
func NewCollector(runner QueryRunner, valType prometheus.ValueType, metricName, query string) *Collector {
	return &Collector{
		runner:     runner,
		metricName: metricName,
		query:      query,
		valType:    valType,
		descs:      nil,
		metrics:    nil,
		mux:        sync.Mutex{},
	}
}

// Describe satisfies the prometheus.Collector interface. Describe is called
// immediately after registering the collector.
func (col *Collector) Describe(ch chan<- *prometheus.Desc) {
	if col.descs == nil {
		// TODO: collect metrics for query exec time.
		col.descs = make(map[string]*prometheus.Desc, 1)
		col.Update()
		col.setDesc()
	}
	// NOTE: if Update returns no metrics, this will fail.
	for _, desc := range col.descs {
		ch <- desc
	}
}

// Collect satisfies the prometheus.Collector interface. Collect reports values
// from cached metrics.
func (col *Collector) Collect(ch chan<- prometheus.Metric) {
	col.mux.Lock()
	// Get reference to current metrics slice to allow Update to run concurrently.
	metrics := col.metrics
	col.mux.Unlock()

	for i := range col.metrics {
		for k, desc := range col.descs {
			ch <- prometheus.MustNewConstMetric(
				desc, col.valType, metrics[i].values[k], metrics[i].labelValues...)
		}
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
		for k, _ := range col.metrics[0].values {
			// TODO: allow passing meaningful help text.
			col.descs[k] = prometheus.NewDesc(col.metricName+k, "help text", col.metrics[0].labelKeys, nil)
		}
	} else {
		// TODO: this is a problem.
		return
	}
}

// bq implements the prometheus.Collector interface for bigquery.
package bq

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	runner QueryRunner
	name   string
	query  string

	valType prometheus.ValueType
	desc    *prometheus.Desc

	metrics []Metric
	mux     sync.Mutex
}

// NewCollector creates a new BigQuery Collector instance.
func NewCollector(runner QueryRunner, valType prometheus.ValueType, metricName, query string) *Collector {
	return &Collector{
		runner:  runner,
		name:    metricName,
		query:   query,
		valType: valType,
		desc:    nil,
		metrics: nil,
		mux:     sync.Mutex{},
	}
}

// Describe satisfies the prometheus.Collector interface. Describe is called
// immediately after registering the collector.
func (bq *Collector) Describe(ch chan<- *prometheus.Desc) {
	if bq.desc == nil {
		// TODO: collect metrics for query exec time.
		bq.Update()
		bq.setDesc()
	}
	// NOTE: if Update returns no metrics, this will fail.
	ch <- bq.desc
}

// Collect satisfies the prometheus.Collector interface. Collect reports values
// from cached metrics.
func (bq *Collector) Collect(ch chan<- prometheus.Metric) {
	bq.mux.Lock()
	defer bq.mux.Unlock()

	for i := range bq.metrics {
		ch <- prometheus.MustNewConstMetric(
			bq.desc, bq.valType, bq.metrics[i].value, bq.metrics[i].values...)
	}
}

// String satisfies the Stringer interface. String returns the metric name.
func (bq *Collector) String() string {
	return bq.name
}

// Update runs the collector query and atomically updates the cached metrics.
// Update is called automaticlly after the collector is registered.
func (bq *Collector) Update() error {
	metrics, err := bq.runner.Query(bq.query)
	if err != nil {
		return err
	}
	// Swap the cached metrics.
	bq.mux.Lock()
	defer bq.mux.Unlock()
	bq.metrics = metrics
	return nil
}

func (bq *Collector) setDesc() {
	// The query may return no results.
	if len(bq.metrics) > 0 {
		// TODO: allow passing meaningful help text.
		bq.desc = prometheus.NewDesc(bq.name, "help text", bq.metrics[0].labels, nil)
	} else {
		// TODO: this is a problem.
		return
	}
}

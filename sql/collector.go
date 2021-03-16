// Package sql implements the prometheus.Collector interface for bigquery.
package sql

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/m-lab/go/logx"

	"github.com/prometheus/client_golang/prometheus"
)

// Metric holds raw data from query results needed to create a prometheus.Metric.
type Metric struct {
	LabelKeys   []string
	LabelValues []string
	Values      map[string]float64
}

// NewMetric creates a Metric with given values.
func NewMetric(labelKeys []string, labelValues []string, values map[string]float64) Metric {
	return Metric{
		LabelKeys:   labelKeys,
		LabelValues: labelValues,
		Values:      values,
	}
}

// QueryRunner defines the interface used to run a query and return an array of metrics.
type QueryRunner interface {
	Query(q string) ([]Metric, error)
}

// Collector manages a prometheus.Collector for queries performed by a QueryRunner.
type Collector struct {
	// runner must be a QueryRunner instance for collecting metrics.
	runner QueryRunner
	// metricName is the base name for prometheus metrics created for this query.
	metricName string
	// query contains the standardSQL query.
	query string
	// lastRun represents the last time the QueryRunner was executed, as unix timestamp in seconds
	lastRun int64
	// minInterval is the minimun interval in seconds between two runs of this Collector
	minInterval int

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

// fetchMinInterval extracts the minimun interval time set on query's file
func fetchMinInterval(queryString string) int {
	minIntervalArg := "--min-interval="
	lines := strings.Split(queryString, "\n")
	for i := range lines {
		line := lines[i]
		if strings.Contains(line, minIntervalArg) {
			var minInterval int
			_, err := fmt.Sscanf(line, minIntervalArg+"%d", &minInterval)
			if err != nil {
				log.Println("Error trying to extract min-inteval from:", line)
			}
			return minInterval
		}
	}
	return 0
}

// NewCollector creates a new BigQuery Collector instance.
func NewCollector(runner QueryRunner, valType prometheus.ValueType, metricName, query string) *Collector {
	return &Collector{
		runner:      runner,
		metricName:  metricName,
		query:       query,
		valType:     valType,
		descs:       nil,
		metrics:     nil,
		mux:         sync.Mutex{},
		minInterval: fetchMinInterval(query),
		lastRun:     -1,
	}
}

// Describe satisfies the prometheus.Collector interface. Describe is called
// immediately after registering the collector.
func (col *Collector) Describe(ch chan<- *prometheus.Desc) {
	logx.Debug.Println("Describe:", time.Now())
	if col.descs == nil {
		// TODO: collect metrics for query exec time.
		col.descs = make(map[string]*prometheus.Desc, 1)
		err := col.Update()
		if err != nil {
			log.Println(err)
		}
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
	logx.Debug.Println("Collect:", time.Now())
	col.mux.Lock()
	// Get reference to current metrics slice to allow Update to run concurrently.
	metrics := col.metrics
	col.mux.Unlock()

	for i := range col.metrics {
		for k, desc := range col.descs {
			logx.Debug.Printf("%s labels:%#v values:%#v",
				col.metricName, metrics[i].LabelValues, metrics[i].Values[k])
			ch <- prometheus.MustNewConstMetric(
				desc, col.valType, metrics[i].Values[k], metrics[i].LabelValues...)
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
	now := time.Now().Unix()
	// Verify if the minumun interval is reached
	if now > col.lastRun+int64(col.minInterval) {
		logx.Debug.Println("Update:", col.metricName)
		col.lastRun = now
		metrics, err := col.runner.Query(col.query)
		if err != nil {
			logx.Debug.Println("Failed to run query:", err)
			return err
		}
		// Swap the cached metrics.
		col.mux.Lock()
		defer col.mux.Unlock()
		// Replace slice reference with new value returned from Query. References
		// to the previous value of col.metrics are not affected.
		col.metrics = metrics
	} else {
		logx.Debug.Println("Minimun interval not reached:", now-col.lastRun, "/", col.minInterval)
	}
	return nil
}

func (col *Collector) setDesc() {
	// The query may return no results.
	if len(col.metrics) > 0 {
		for k := range col.metrics[0].Values {
			// TODO: allow passing meaningful help text.
			col.descs[k] = prometheus.NewDesc(col.metricName+k, "help text", col.metrics[0].LabelKeys, nil)
		}
	}
}

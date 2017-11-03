// bigquery_exporter runs structured bigquery SQL and converts the results into
// prometheus metrics. bigquery_exporter can process multiple queries.
// Because BigQuery queries can have long run times and high cost, Query results
// are cached and updated every refresh interval, not on every scrape of
// prometheus metrics.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/m-lab/prometheus-bigquery-exporter/bq"

	flag "github.com/spf13/pflag"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	valueTypes   = []string{}
	querySources = []string{}
	project      = flag.String("project", "", "GCP project name.")
	port         = flag.String("port", "9050", "Exporter port.")
	refresh      = flag.Duration("refresh", 5*time.Minute, "Interval between updating metrics.")
)

func init() {
	flag.StringArrayVar(&valueTypes, "type", nil, "Name of the prometheus value type, e.g. 'counter' or 'gauge'.")
	flag.StringArrayVar(&querySources, "query", nil, "Name of file with query string.")

	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// sleepUntilNext finds the nearest future time that is a multiple of the given
// duration and sleeps until that time.
func sleepUntilNext(d time.Duration) {
	next := time.Now().Truncate(d).Add(d)
	time.Sleep(time.Until(next))
}

// fileToMetric extracts the base file name to use as a prometheus metric name.
func fileToMetric(filename string) string {
	fname := filepath.Base(filename)
	return strings.TrimSuffix(fname, filepath.Ext(fname))
}

// createCollector creates a bq.Collector initialized with the BQ query
// contained in filename. The returned collector should be registered with
// prometheus.Register.
func createCollector(client *bigquery.Client, filename, typeName string, vars map[string]string) (*bq.Collector, error) {
	queryBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var v prometheus.ValueType
	if typeName == "counter" {
		v = prometheus.CounterValue
	} else if typeName == "gauge" {
		v = prometheus.GaugeValue
	} else {
		v = prometheus.UntypedValue
	}

	// TODO: use to text/template
	query := string(queryBytes)
	query = strings.Replace(query, "UNIX_START_TIME", vars["UNIX_START_TIME"], -1)
	query = strings.Replace(query, "REFRESH_RATE_SEC", vars["REFRESH_RATE_SEC"], -1)

	c := bq.NewCollector(bq.NewQueryRunner(client), v, fileToMetric(filename), string(query))

	return c, nil
}

// updatePeriodically runs in an infinite loop, and updates registered
// collectors every refresh period.
func updatePeriodically(unregistered chan *bq.Collector, refresh time.Duration) {
	var collectors = []*bq.Collector{}

	// Attempt to register all unregistered collectors.
	if len(unregistered) > 0 {
		collectors = append(collectors, tryRegister(unregistered)...)
	}
	for sleepUntilNext(refresh); ; sleepUntilNext(refresh) {
		log.Printf("Starting a new round at: %s", time.Now())
		for i := range collectors {
			log.Printf("Running query for %s", collectors[i])
			collectors[i].Update()
			log.Printf("Done")
		}
		if len(unregistered) > 0 {
			collectors = append(collectors, tryRegister(unregistered)...)
		}
	}
}

// tryRegister attempts to prometheus.Register every bq.Collectors queued in
// unregistered. Any collectors that fail are placed back on the channel. All
// successfully registered collectors are returned.
func tryRegister(unregistered chan *bq.Collector) []*bq.Collector {
	var registered = []*bq.Collector{}
	count := len(unregistered)
	for i := 0; i < count; i++ {
		// Take collector off of channel.
		c := <-unregistered

		// Try to register this collector.
		err := prometheus.Register(c)
		if err != nil {
			// Registration failed, so place collector back on channel.
			unregistered <- c
			continue
		}
		log.Printf("Registered %s", c)
		registered = append(registered, c)
	}
	return registered
}

func main() {
	flag.Parse()

	if len(querySources) != len(valueTypes) {
		log.Fatal("You must provide a --type flag for every --query source.")
	}

	// Create a channel with capacity for all collectors.
	unregistered := make(chan *bq.Collector, len(querySources))

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, *project)
	if err != nil {
		log.Fatal(err)
	}

	vars := map[string]string{
		"UNIX_START_TIME":  fmt.Sprintf("%d", time.Now().UTC().Unix()),
		"REFRESH_RATE_SEC": fmt.Sprintf("%d", int(refresh.Seconds())),
	}
	for i := range querySources {
		c, err := createCollector(client, querySources[i], valueTypes[i], vars)
		if err != nil {
			log.Printf("Failed to create collector %s: %s", querySources[i], err)
			continue
		}
		// Store collector in channel.
		unregistered <- c
	}

	go updatePeriodically(unregistered, *refresh)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

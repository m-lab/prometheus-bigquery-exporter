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
	refresh      = flag.Duration("refresh", 5*time.Minute, "Number of seconds between refreshing.")
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

// registerCollector
func createCollector(typeName, filename string, refresh time.Duration) (*bq.Collector, error) {
	queryBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, *project)
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

	query := string(queryBytes)
	query = strings.Replace(query, "UNIX_START_TIME", fmt.Sprintf("%d", time.Now().UTC().Unix()), -1)
	query = strings.Replace(query, "REFRESH_RATE_SEC", fmt.Sprintf("%d", int(refresh.Seconds())), -1)

	c := bq.NewCollector(bq.NewQueryRunner(client), v, fileToMetric(filename), string(query))

	return c, nil
}

func updatePeriodically(collectors, unregistered []*bq.Collector, refresh time.Duration) {
	var registered = []*bq.Collector{}

	if len(unregistered) > 0 {
	}
	tryRegister(unregistered)
		err := prometheus.Register(c)

	for sleepUntilNext(refresh); ; sleepUntilNext(refresh) {
		log.Printf("Starting a new round at: %s", time.Now())
		for i := range collectors {
			log.Printf("Running query for %s", collectors[i])
			collectors[i].Update()
			log.Printf("Done")
		}
	}
}

func tryRegister(unregistered []*bq.Collector) error {
		// Attempt to register collector. If it fails, retry later.
	}
}

func main() {
	flag.Parse()
	var unregistered = []*bq.Collector{}

	if len(querySources) != len(valueTypes) {
		log.Fatal("You must provide a --type flag for every --query source.")
	}

	for i := range querySources {
		c, err := createCollector(valueTypes[i], querySources[i], *refresh)
		if err != nil {
			log.Printf("Failed to create collector %s: %s", querySources[i], err)
			continue
		}
		unregistered = append(unregistered, c)
	}

	go updatePeriodically(unregistered, *refresh)

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9393", nil))
}

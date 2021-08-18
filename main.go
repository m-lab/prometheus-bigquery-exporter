// bigquery_exporter runs structured bigquery SQL and converts the results into
// prometheus metrics. bigquery_exporter can process multiple queries.
// Because BigQuery queries can have long run times and high cost, Query results
// are cached and updated every refresh interval, not on every scrape of
// prometheus metrics.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/logx"
	"github.com/m-lab/go/prometheusx"
	"github.com/m-lab/go/rtx"
	"github.com/m-lab/prometheus-bigquery-exporter/internal/config"
	"github.com/m-lab/prometheus-bigquery-exporter/internal/setup"
	"github.com/m-lab/prometheus-bigquery-exporter/query"
	"github.com/m-lab/prometheus-bigquery-exporter/sql"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	//counterSources = flagx.StringArray{}
	//gaugeSources   = flagx.StringArray{}
	//project    = flag.String("project", "", "GCP project name.")
	configFile = flag.String("config", "config.yaml", "Configuration file name")
	refresh    = flag.Duration("refresh", 5*time.Minute, "Interval between updating metrics.")
)

func init() {

	//flag.Var(&counterSources, "counter-query", "Name of file containing a counter query.")
	//flag.Var(&gaugeSources, "gauge-query", "Name of file containing a gauge query.")

	// Port registered at https://github.com/prometheus/prometheus/wiki/Default-port-allocations
	*prometheusx.ListenAddress = ":9348"
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

// fileToQuery reads the content of the given file and returns the query with template values repalced with those in vars.
func fileToQuery(filename string, vars map[string]string) string {
	queryBytes, err := ioutil.ReadFile(filename)
	rtx.Must(err, "Failed to open %q", filename)

	q := string(queryBytes)
	q = strings.Replace(q, "UNIX_START_TIME", vars["UNIX_START_TIME"], -1)
	q = strings.Replace(q, "REFRESH_RATE_SEC", vars["REFRESH_RATE_SEC"], -1)
	return q
}

func reloadRegisterUpdate(client *bigquery.Client, GaugeFiles []setup.File, CounterFiles []setup.File, vars map[string]string, isConfigFileModified bool) {
	var wg sync.WaitGroup
	for i := range GaugeFiles {
		wg.Add(1)
		go func(f *setup.File, isConfigFileModified bool) {
			modified, err := f.IsModified()
			if (modified && err == nil) || isConfigFileModified {
				c := sql.NewCollector(
					newRunner(client), prometheus.GaugeValue,
					fileToMetric(f.Name), fileToQuery(f.Name, vars))

				log.Println("Registering:", fileToMetric(f.Name))
				// NOTE: prometheus collector registration will fail when a file
				// uses the same name but changes the metrics reported. Because
				// this cannot be recovered, we use rtx.Must to exit and allow
				// the runtime environment to restart.
				rtx.Must(f.Register(c), "Failed to register collector: aborting")
			} else {
				start := time.Now()
				err = f.Update()
				log.Println("Updating:", fileToMetric(f.Name), time.Since(start))
			}
			if err != nil {
				log.Println("Error:", f.Name, err)
			}
			wg.Done()
		}(&GaugeFiles[i], isConfigFileModified)
	}
	wg.Wait()
	for i := range CounterFiles {
		wg.Add(1)
		go func(f *setup.File, isConfigFileModified bool) {
			modified, err := f.IsModified()
			if (modified && err == nil) || isConfigFileModified {
				c := sql.NewCollector(
					newRunner(client), prometheus.CounterValue,
					fileToMetric(f.Name), fileToQuery(f.Name, vars))

				log.Println("Registering:", fileToMetric(f.Name))
				err = f.Register(c)
			} else {
				log.Println("Updating:", fileToMetric(f.Name))
				err = f.Update()
			}
			if err != nil {
				log.Println("Error:", f.Name, err)
			}
			wg.Done()
		}(&CounterFiles[i], isConfigFileModified)
	}
	wg.Wait()
}

var mainCtx, mainCancel = context.WithCancel(context.Background())
var newRunner = func(client *bigquery.Client) sql.QueryRunner {
	return query.NewBQRunner(client)
}

func main() {

	flag.Parse()
	rtx.Must(flagx.ArgsFromEnv(flag.CommandLine), "Could not get args from env")

	if configFile == nil {
		fmt.Printf("Undefined config file path")
		os.Exit(1)
	}

	cfg := initConfig(*configFile)

	srv := prometheusx.MustServeMetrics()
	defer srv.Shutdown(mainCtx)

	GaugeFiles := toFiles(cfg.GetGaugeFiles())
	CounterFiles := toFiles(cfg.GetCounterFiles())

	client, err := bigquery.NewClient(mainCtx, cfg.Project)
	rtx.Must(err, "Failed to allocate a new bigquery.Client")
	vars := map[string]string{
		"UNIX_START_TIME":  fmt.Sprintf("%d", time.Now().UTC().Unix()),
		"REFRESH_RATE_SEC": fmt.Sprintf("%d", int(refresh.Seconds())),
	}

	for mainCtx.Err() == nil {

		isModified, err := cfg.IsModified()
		if err != nil {
			logx.Debug.Fatalf("Something wrong during configuration reload: %s", err.Error())
			os.Exit(1)
		}
		if isModified {

			logx.Debug.Printf("Start reload configuration")
			cfg = initConfig(*configFile)
			GaugeFiles = toFiles(cfg.GetGaugeFiles())
			CounterFiles = toFiles(cfg.GetCounterFiles())
			logx.Debug.Printf("Configuration reload completed successfully: %+v", cfg)
		}

		reloadRegisterUpdate(client, GaugeFiles, CounterFiles, vars, isModified)
		sleepUntilNext(*refresh)
	}
}

func toFiles(paths []string) []setup.File {
	files := make([]setup.File, len(paths))
	for i := range paths {
		files[i].Name = paths[i]
	}
	return files
}

func initConfig(configPath string) *config.Config {

	cfg, err := config.ReadConfigFile(configPath)
	if err != nil {
		logx.Debug.Fatalf("%s", err.Error())
		os.Exit(1)
	}

	logx.Debug.Printf("Configuration unmarshalled successfully: %+v", cfg)
	return cfg
}

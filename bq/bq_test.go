package bq

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type fakeQueryRunner struct {
	metrics []Metric
}

func (qr *fakeQueryRunner) Query(query string) ([]Metric, error) {
	return qr.metrics, nil
}

func TestCollector(t *testing.T) {
	metrics := []Metric{
		Metric{
			labelKeys:   []string{"key"},
			labelValues: []string{"thing"},
			values:      map[string]float64{"": 1.1},
		},
		Metric{
			labelKeys:   []string{"key"},
			labelValues: []string{"thing2"},
			values:      map[string]float64{"": 2.1},
		},
	}
	expectedMetrics := []string{
		`fake_metric{key="thing"} 1.1`,
		`fake_metric{key="thing2"} 2.1`,
	}
	c := NewCollector(
		&fakeQueryRunner{metrics}, prometheus.GaugeValue, "fake_metric", "-- not used")

	// NOTE: prometheus.Desc and prometheus.Metric are opaque interfaces that do
	// not allow introspection. But, we know how many to expect, so check the
	// counts added to the channels.
	chDesc := make(chan *prometheus.Desc, 2)
	chCol := make(chan prometheus.Metric, 2)

	c.Describe(chDesc)
	c.Collect(chCol)

	close(chDesc)
	close(chCol)

	if len(chDesc) != 1 {
		t.Fatal("want 1 prometheus.Desc, got %d\n", len(chDesc))
	}
	if len(chCol) != 2 {
		t.Fatal("want 2 prometheus.Metric, got %d\n", len(chCol))
	}

	// Normally, we use the default registry via prometheus.Register. Using a
	// custom registry allows us to write clearer tests.
	reg := prometheus.NewRegistry()
	err := reg.Register(c)
	if err != nil {
		t.Fatal("could not register collector.")
	}

	// Read all metrics via the prometheus handler.
	ts := httptest.NewServer(promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	defer ts.Close()

	// Get the raw metrics from the test server handler.
	res, err := http.Get(ts.URL)
	if err != nil {
		log.Fatal(err)
	}
	rawMetrics, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Check for the expected metrics in the rawMetrics.
	lines := strings.Split(string(rawMetrics), "\n")
	for _, expected := range expectedMetrics {
		found := false
		for _, line := range lines {
			if strings.Contains(line, expected) {
				found = true
			}
		}
		if !found {
			t.Error("Did not find expected metric: %s", expected)
		}
	}

}

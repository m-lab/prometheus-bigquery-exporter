package setup

import (
	"fmt"
	"testing"
	"time"

	"github.com/m-lab/go/rtx"
	"github.com/m-lab/prometheus-bigquery-exporter/sql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/afero"
)

func TestFile_IsModified(t *testing.T) {
	// Override the package afero.OsFs with a local memory fs with a single file.
	fs = afero.NewMemMapFs()
	fs.Create("localfile")

	// Create a fake stat object that's before current time of localfile.
	fs.Create("fakestat")
	before := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	fs.Chtimes("fakestat", before, before)
	s, err := fs.Stat("fakestat")
	rtx.Must(err, "Failed to stat filesystem")

	tests := []struct {
		name    string
		file    *File
		want    bool
		wantErr bool
	}{
		{
			name: "success-first-run",
			file: &File{
				Name: "localfile",
			},
			want: true,
		},
		{
			name: "success-second-run",
			file: &File{
				Name: "localfile",
				stat: s,
			},
			want: true,
		},
		{
			name: "error-missing-file",
			file: &File{
				Name: "file-not-found",
				stat: s,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.file.IsModified()
			if (err != nil) != tt.wantErr {
				t.Errorf("File.IsModified(%q) error = %v, wantErr %v", tt.file.Name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("File.IsModified() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeRunner struct{}

func (f *fakeRunner) Query(query string) ([]sql.Metric, error) {
	return nil, fmt.Errorf("Fake failure")
}

func TestFile_Update(t *testing.T) {
	tests := []struct {
		name    string
		c       *sql.Collector
		wantErr bool
	}{
		{
			name: "success",
			c:    nil,
		},
		{
			name:    "error-from-update",
			c:       sql.NewCollector(&fakeRunner{}, prometheus.GaugeValue, "foo", ""),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{
				Name: "example",
				c:    tt.c,
			}
			if err := f.Update(); (err != nil) != tt.wantErr {
				t.Errorf("File.Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type fakeRegister struct {
	metric sql.Metric
}

func (f *fakeRegister) Query(query string) ([]sql.Metric, error) {
	return []sql.Metric{f.metric}, nil
}

func TestFile_Register(t *testing.T) {
	fr := &fakeRegister{
		metric: sql.NewMetric([]string{}, []string{}, map[string]float64{"": 1.23}),
	}
	x := sql.NewCollector(fr, prometheus.GaugeValue, "foo", "")
	tests := []struct {
		name          string
		fileCollector *sql.Collector
		newCollector  *sql.Collector
		wantErr       bool
	}{
		{
			// Register should succeed.
			name:         "register-success",
			newCollector: x,
		},
		{
			// Try to register the same collector should return an error.
			name:         "register-returns-error",
			newCollector: x,
			wantErr:      true,
		},
		{
			// Unregister the same one registered above.
			name:          "unregister-success",
			fileCollector: x,
			newCollector:  x,
		},
		{
			// Try to unregister a collector that was never registered.
			name:          "unregister-returns-error",
			fileCollector: sql.NewCollector(&fakeRunner{}, prometheus.GaugeValue, "foo", ""),
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{
				Name: "example",
				c:    tt.fileCollector,
			}
			if err := f.Register(tt.newCollector); (err != nil) != tt.wantErr {
				t.Errorf("File.Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

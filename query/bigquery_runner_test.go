package query

import (
	"fmt"
	"math"
	"reflect"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-lab/prometheus-bigquery-exporter/sql"

	"github.com/m-lab/go/cloud/bqfake"
)

func TestRowToMetric(t *testing.T) {
	tests := []struct {
		name    string
		row     map[string]bigquery.Value
		metric  sql.Metric
		wantNaN bool
	}{
		{
			name: "Extract labels",
			row: map[string]bigquery.Value{
				"machine": "mlab1.foo01.measurement-lab.org",
				"value":   1.0,
			},
			metric: sql.Metric{
				LabelKeys:   []string{"machine"},
				LabelValues: []string{"mlab1.foo01.measurement-lab.org"},
				Values:      map[string]float64{"": 1.0},
			},
		},
		{
			name: "No labels",
			row: map[string]bigquery.Value{
				"value": 1.1,
			},
			metric: sql.Metric{
				LabelKeys:   nil,
				LabelValues: nil,
				Values:      map[string]float64{"": 1.1},
			},
		},
		{
			name: "Integer value",
			row: map[string]bigquery.Value{
				"value": int64(10),
			},
			metric: sql.Metric{
				LabelKeys:   nil,
				LabelValues: nil,
				Values:      map[string]float64{"": 10},
			},
		},
		{
			name: "Multiple values",
			row: map[string]bigquery.Value{
				"value_foo": int64(10),
				"value_bar": int64(20),
			},
			metric: sql.Metric{
				LabelKeys:   nil,
				LabelValues: nil,
				Values:      map[string]float64{"_foo": 10, "_bar": 20},
			},
		},
		{
			name: "Bad label values are converted to strings",
			row: map[string]bigquery.Value{
				"name":  3.0, // should be a string.
				"value": 2.1,
			},
			metric: sql.Metric{
				LabelKeys:   []string{"name"},
				LabelValues: []string{"invalid string"}, // converted to a string.
				Values:      map[string]float64{"": 2.1},
			},
		},
		{
			name: "NaN value",
			row: map[string]bigquery.Value{
				"value": "this-is-NaN",
			},
			metric: sql.Metric{
				Values: map[string]float64{"": math.NaN()},
			},
			wantNaN: true,
		},
	}

	for _, test := range tests {
		m := rowToMetric(test.row)
		if !test.wantNaN && !reflect.DeepEqual(m, test.metric) {
			t.Errorf("Failed to convert row to metric. want %#v; got %#v", test.metric, m)
		}
		if test.wantNaN && !math.IsNaN(m.Values[""]) {
			t.Errorf("Failed to convert row to metric. want %#v; got %#v", test.metric, m)
		}
	}
}

type fakeQuery struct {
	err  error
	rows []map[string]bigquery.Value
}

func (f *fakeQuery) Query(q string, visit func(row map[string]bigquery.Value) error) error {
	if f.err != nil {
		return f.err
	}
	for i := range f.rows {
		err := visit(f.rows[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func TestBQRunner_Query(t *testing.T) {
	tests := []struct {
		name    string
		runner  runner
		want    []sql.Metric
		wantErr bool
	}{
		{
			name: "okay",
			runner: &fakeQuery{
				rows: []map[string]bigquery.Value{
					{
						"value_name": 1.23,
					},
				},
			},
			want: []sql.Metric{
				sql.NewMetric(nil, nil, map[string]float64{"_name": 1.23}),
			},
		},
		{
			name: "query-error",
			runner: &fakeQuery{
				err: fmt.Errorf("Fake query error"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qr := &BQRunner{
				runner: tt.runner,
			}
			got, err := qr.Query("select * from `fake-table`")
			if (err != nil) != tt.wantErr {
				t.Errorf("BQRunner.Query() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BQRunner.Query() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNewBQRunner(t *testing.T) {
	NewBQRunner(nil)
}

func TestBigQueryImpl_Query(t *testing.T) {
	tests := []struct {
		name    string
		config  bqfake.QueryConfig[map[string]bigquery.Value]
		query   string
		visit   func(row map[string]bigquery.Value) error
		wantErr bool
	}{
		{
			name: "success-iteration",
			config: bqfake.QueryConfig[map[string]bigquery.Value]{
				RowIteratorConfig: bqfake.RowIteratorConfig[map[string]bigquery.Value]{
					Rows: []map[string]bigquery.Value{{"value": 1.234}},
				},
			},
			visit: func(row map[string]bigquery.Value) error {
				return nil
			},
		},
		{
			name: "visit-error",
			config: bqfake.QueryConfig[map[string]bigquery.Value]{
				RowIteratorConfig: bqfake.RowIteratorConfig[map[string]bigquery.Value]{
					Rows: []map[string]bigquery.Value{{"value": 1.234}},
				},
			},
			visit: func(row map[string]bigquery.Value) error {
				return fmt.Errorf("Fake visit error")
			},
			wantErr: true,
		},
		{
			name: "read-error",
			config: bqfake.QueryConfig[map[string]bigquery.Value]{
				ReadErr: fmt.Errorf("This is a fake read error"),
			},
			wantErr: true,
		},
		{
			name: "iterator-error",
			config: bqfake.QueryConfig[map[string]bigquery.Value]{
				RowIteratorConfig: bqfake.RowIteratorConfig[map[string]bigquery.Value]{
					IterErr: fmt.Errorf("This is a fake iterator error"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := bqfake.NewQueryReadClient(tt.config)
			b := &bigQueryImpl{
				Client: client,
			}
			if err := b.Query(tt.query, tt.visit); (err != nil) != tt.wantErr {
				t.Errorf("bigQueryImpl.Query() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

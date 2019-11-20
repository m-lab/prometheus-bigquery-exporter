package bqiface

import (
	"fmt"
	"testing"

	"github.com/m-lab/go/cloud/bqfake"

	"cloud.google.com/go/bigquery"
	"github.com/GoogleCloudPlatform/google-cloud-go-testing/bigquery/bqiface"
)

func TestBigQueryImpl_Query(t *testing.T) {
	tests := []struct {
		name    string
		Client  bqiface.Client
		query   string
		visit   func(row map[string]bigquery.Value) error
		wantErr bool
	}{
		{
			name:   "success-iteration",
			Client: bqfake.NewQueryReadClient([]map[string]bigquery.Value{{"value": 1.234}}, nil),
			visit: func(row map[string]bigquery.Value) error {
				return nil
			},
		},
		{
			name:   "visit-error",
			Client: bqfake.NewQueryReadClient([]map[string]bigquery.Value{{"value": 1.234}}, nil),
			visit: func(row map[string]bigquery.Value) error {
				return fmt.Errorf("Fake visit error")
			},
			wantErr: true,
		},
		{
			name:    "read-error",
			Client:  bqfake.NewQueryReadClient(nil, fmt.Errorf("This is a fake read error")),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &BigQueryImpl{
				Client: tt.Client,
			}
			if err := b.Query(tt.query, tt.visit); (err != nil) != tt.wantErr {
				t.Errorf("BigQueryImpl.Query() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

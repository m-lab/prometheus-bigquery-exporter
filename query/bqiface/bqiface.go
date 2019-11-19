package bqiface

import (
	"context"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

type BigQueryImpl struct {
	Client *bigquery.Client
}

func (b *BigQueryImpl) Query(query string, visit func(row map[string]bigquery.Value) error) error {
	q := b.Client.Query(query)
	it, err := q.Read(context.Background())
	if err != nil {
		return err
	}
	var row map[string]bigquery.Value
	for err = it.Next(&row); err != iterator.Done; err = it.Next(&row) {
		err2 := visit(row)
		if err2 != nil {
			return err2
		}
	}
	return nil
}

# prometheus-bigquery-exporter

[![Version](https://img.shields.io/github/tag/m-lab/prometheus-bigquery-exporter.svg)](https://github.com/m-lab/prometheus-bigquery-exporter/releases) [![Build Status](https://travis-ci.org/m-lab/prometheus-bigquery-exporter.svg?branch=master)](https://travis-ci.org/m-lab/prometheus-bigquery-exporter) [![Coverage Status](https://coveralls.io/repos/m-lab/prometheus-bigquery-exporter/badge.svg?branch=master)](https://coveralls.io/github/m-lab/prometheus-bigquery-exporter?branch=master) [![GoDoc](https://godoc.org/github.com/m-lab/prometheus-bigquery-exporter?status.svg)](https://godoc.org/github.com/m-lab/prometheus-bigquery-exporter) [![Go Report Card](https://goreportcard.com/badge/github.com/m-lab/prometheus-bigquery-exporter)](https://goreportcard.com/report/github.com/m-lab/prometheus-bigquery-exporter)

An exporter for converting BigQuery results into Prometheus metrics.

## Limitations: No historical values

Prometheus collects the *current* status of a system as reported by an exporter.
Prometheus then associates the values collected with a timestamp of the time of
collection.

*NOTE:* there is no way to associate historical values with timestamps in the
the past with this exporter!

So, the results of queries run by prometheus-bigquery-exporter should represent
a meaningful value at a fixed point in time relative to the time the query is
made, e.g. total number of tests in a 5 minute window 1 hour ago.

## Query Formatting

The prometheus-bigquery-exporter accepts arbitrary BQ queries. However, the
query results must be structured in a predictable way for the exporter to
successfully interpret and convert it into prometheus metrics.

### Metric names and values

Metric names are derived from the query file name and query value columns.
The bigquery-exporter identifies value columns by looking for column names
that match the pattern: `value([.+])`. All characters in the matching group
`([.+])` are appended to the metric prefix taken from the query file name.
For example:

* Filename: `bq_ndt_test.sql`
* Metric prefix: `bq_ndt_test`
* Column name: `value_count`
* Final metric: `bq_ndt_test_count`

Value columns are required (at least one):

* `value([.+])` - every query must define a result "value". Values must
  be integers or floats. For a query to return multiple values, prefix each
  with "value" and define unique suffixes.

Label columns are optional:

* If there is more than one result row, then the query must also define labels
  to distinguish each value. Every column name that is not "value" will create
  a label on the resulting metric. For example, results with two columns,
  "machine" and "value" would create metrics with labels named "machine" and
  values from the results for that row.

Labels must be strings:

* There is no limit on the number of labels, but you should respect the
  prometheus best practices by limiting label value cardinality.

Duplicate metrics are an error:

* If the query returns multiple rows that are not distinguished by the set of
  labels for each row.

## Example Query

The following query creates a label and groups by each label.

  ```sql
  -- Example data in place of an actual table of values.
  WITH example_data as (
      SELECT "a" as label, 5 as widgets
      UNION ALL
      SELECT "b" as label, 2 as widgets
      UNION ALL
      SELECT "b" as label, 3 as widgets
  )

  SELECT
     label, SUM(widgets) as value
  FROM
     example_data
  GROUP BY
     label
  ```

* Save the sample query to a file named "bq_example.sql".
* Start the exporter:

  ```sh
  prometheus-bigquery-exporter -gauge-query bq_example.sql
  ```

* Visit http://localhost:9348/metrics and you will find metrics like:

  ```txt
    bq_example{label="a"} 5
    bq_example{label="b"} 5
    ...
  ```

## Example Configuration

Typical deployments will be in Kubernetes environment, like GKE.

```sh
# Change to the example directory.
cd example
# Deploy the example query as a configmap and example k8s deployment.
./deploy.sh
```

## Testing

To run the bigquery exporter locally (e.g. with a new query) you can build
and run locally.

Use the following steps:

1. Build the docker image.

  ```sh
  docker build -t bqx-local -f Dockerfile .
  ```

2. Authenticate using your Google account. Both steps are necessary, the
  first to run gcloud commands (which uses user credentials), the second to run
  the bigquery exporter (which uses application default credentials).

  ```sh
  gcloud auth login
  gcloud auth application-default login
  ```

3. Run the image, with fowarded ports and access to gcloud credentials.

  ```sh
  docker run -p 9348:9348 --rm \
    -v $HOME/.config/gcloud:/root/.config/gcloud \
    -v $PWD:/queries -it bqx-local \
      -project=$GCLOUD_PROJECT \
      -gauge-query=/queries/example/config/bq_example.sql
  ```

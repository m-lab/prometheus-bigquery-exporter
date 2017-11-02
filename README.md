# prometheus-bigquery-exporter
An exporter for converting BigQuery results into Prometheus metrics.

# Limitations

## No historical values

Prometheus collects the *current* status of a system as reported by an exporter.
Prometheus then associates the values collected with a timestamp of the time of
collection.

*NOTE:* there is no way to associate historical values with timestamps in the
the past!

So, the results of queries run by prometheus-bigquery-exporter should represent
a meaningful value at a fixed point in time relative to the time the query is
made, e.g. total number of tests in a 5 minute window 1 hour ago.

# Query format

The prometheus-bigquery-exporter accepts arbitrary BQ queries. However, the
query results must be structured in a predictable way for the exporter to
successfully interpret and convert it into prometheus metrics.

Required columns:

 * `value` -- every query result must have a "value". Values should be integers
   or floats.

Optional columns:

 * If there is more than one result row, then the query must also define labels
   to distinguish each value. Every column name that is not "value" will create
   a label on the resulting metric. For example, results with two columns,
   "machine" and "value" would create metrics with labels named "machine" and
   values from the results for that row.

   Labels should be strings.

   There is no limit on the number of labels, but you should respect the
   prometheus best practices by limiting label value cardinality.

## Example query

The following query creates a "machine" label and counts the number of tests

```
# TODO: replace with query using views.
# TODO: replace with standard SQL syntax.
SELECT
    -- All columns not named "value" are added as metric labels.
    CONCAT(REPLACE(REGEXP_EXTRACT(task_filename,
        r'gs://.*-(mlab[1-4]-[a-z]{3}[0-9]+)-ndt.*.tgz'), "-", "."),
        ".measurement-lab.org") AS label_machine,

    -- All queries must have a single column named "value"
    count(*) as value

FROM
    [measurement-lab:public.ndt]

GROUP BY label_machine
ORDER BY value
```

Save the sample query to a file named "ndt_test_cound.sql". The metric name is
derived from the file name. Start the exporter:

```
    bq_exporter --query counter=ndt_test_count.sql
```

Visit http://localhost:9393/metrics and you will find metrics like:

```
    ndt_test_count{machine="mlab1.foo01.measurement-lab.org"} 100
    ndt_test_count{machine="mlab2.foo01.measurement-lab.org"} 200
    ...
```

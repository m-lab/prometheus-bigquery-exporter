FROM golang:1.12
ADD . /go/src/github.com/m-lab/prometheus-bigquery-exporter
RUN go get -v github.com/m-lab/prometheus-bigquery-exporter/cmd/bigquery_exporter
ENTRYPOINT ["/go/bin/bigquery_exporter"]

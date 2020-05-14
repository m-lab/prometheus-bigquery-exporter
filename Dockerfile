FROM quay.io/prometheus/golang-builder:1.14.2-main as builder

ADD . /go/src/github.com/m-lab/prometheus-bigquery-exporter
WORKDIR /go/src/github.com/m-lab/prometheus-bigquery-exporter

RUN make

FROM quay.io/prometheus/busybox:glibc

COPY --from=builder /go/bin/prometheus-bigquery-exporter /bin/prometheus-bigquery-exporter

EXPOSE 9348

ENTRYPOINT  [ "/bin/prometheus-bigquery-exporter" ]

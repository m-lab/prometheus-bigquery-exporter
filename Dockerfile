FROM quay.io/prometheus/golang-builder as builder

ADD . /go/src/github.com/m-lab/prometheus-bigquery-exporter
WORKDIR /go/src/github.com/m-lab/prometheus-bigquery-exporter

RUN make

FROM quay.io/prometheus/busybox:glibc

COPY --from=builder /go/src/github.com/m-lab/prometheus-bigquery-exporter/prometheus-bigquery-exporter /bin/prometheus-bigquery-exporter

EXPOSE 9348

ENTRYPOINT  [ "/bin/prometheus-bigquery-exporter" ]

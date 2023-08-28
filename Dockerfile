FROM golang:1.20 as builder
ADD . /go/src/github.com/m-lab/prometheus-bigquery-exporter
WORKDIR /go/src/github.com/m-lab/prometheus-bigquery-exporter
ENV CGO_ENABLED 0
RUN go vet && \
    go get -t . && \
    go install .

FROM alpine:3.15
COPY --from=builder /go/bin/prometheus-bigquery-exporter /bin/prometheus-bigquery-exporter
EXPOSE 9348
ENTRYPOINT  [ "/bin/prometheus-bigquery-exporter" ]

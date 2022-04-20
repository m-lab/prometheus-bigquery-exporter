FROM golang:alpine3.15 as builder
ADD . /go/src/github.com/m-lab/prometheus-bigquery-exporter
WORKDIR /go/src/github.com/m-lab/prometheus-bigquery-exporter
RUN apk add gcc libc-dev git
RUN go vet
RUN go get -t .
RUN go install .

FROM alpine:3.15
COPY --from=builder /go/bin/prometheus-bigquery-exporter /bin/prometheus-bigquery-exporter
EXPOSE 9348
ENTRYPOINT  [ "/bin/prometheus-bigquery-exporter" ]

#!/bin/bash
#
# deploy-example.sh applies an example bigquery exporter to the currently
# selected k8s cluster.
#
# Example:
#
# ./deploy-example.sh

set -x
set -e
set -u

# Apply the bigquery exporter configurations.
kubectl create configmap bigquery-exporter-config \
    --from-file=example/config \
    --dry-run=client -o json | kubectl apply -f -

kubectl apply -f example/bigquery.yml

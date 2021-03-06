#!/bin/bash
#
# deploy.sh applies an example bigquery exporter to the currently
# selected k8s cluster.
#
# Example:
#
# ./deploy.sh

set -x
set -e
set -u

# Apply the bigquery exporter configurations.
kubectl create configmap bigquery-exporter-config \
    --from-file=config \
    --dry-run=client -o json | kubectl apply -f -

kubectl apply -f bigquery.yml

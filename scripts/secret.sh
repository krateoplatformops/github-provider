#!/bin/bash

set -o allexport
source .env
set +o allexport

kubectl create secret generic github-secret \
    --namespace=demo-system \
    --from-literal=token=${TOKEN} || true


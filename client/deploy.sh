#!/usr/bin/env bash

kubectl create configmap envoy-config-client --from-file=bootstrap=envoy.yaml -o yaml --dry-run > envoy-configmap.yaml
kubectl apply -f mesh-client-deployment.yaml
kubectl apply -f envoy-configmap.yaml
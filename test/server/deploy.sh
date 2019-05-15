#!/usr/bin/env bash

kubectl create configmap envoy-config-server --from-file=bootstrap=envoy.yaml -o yaml --dry-run > envoy-configmap.yaml
kubectl apply -f mesh-server-deployment.yaml
kubectl apply -f mesh-server-service.yaml
kubectl apply -f envoy-configmap.yaml
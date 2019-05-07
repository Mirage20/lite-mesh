#!/usr/bin/env bash

kubectl delete -f mesh-server-deployment.yaml
kubectl delete -f mesh-server-service.yaml
kubectl delete -f envoy-configmap.yaml
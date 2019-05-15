#!/usr/bin/env bash

kubectl delete -f mesh-client-deployment.yaml
kubectl delete -f envoy-configmap.yaml
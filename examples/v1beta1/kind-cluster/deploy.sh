#!/bin/bash

# Copyright 2021 The Kubeflow Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script is used to deploy Kind cluster with Katib standalone components.
set -e

# Verify that appropriate tools are installed.
if [[ ! $(which docker) ]]; then
  echo "Unable to find Docker"
  echo "To install Docker, please follow this guide: https://docs.docker.com/get-docker"
  exit 1
fi

if [[ ! $(which kind) ]]; then
  echo "Unable to find Kind"
  echo "To install Kind, please follow this guide: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
  exit 1
fi

if [[ ! $(which kubectl) ]]; then
  echo "Unable to find kubectl"
  echo "To install kubectl, please follow this guide: https://kubernetes.io/docs/tasks/tools/#kubectl"
  exit 1
fi

# Step 1. Create Kind cluster with Kubernetes v1.21.1
kind create cluster --image kindest/node:v1.21.1
echo -e "\nKind cluster has been created\n"

# Step 2. Set context for kubectl
kubectl config use-context kind-kind

# Step 3. Wait until Kubernetes Nodes will be ready.
TIMEOUT=30m
kubectl wait --for=condition=ready --timeout=${TIMEOUT} node kind-control-plane

kubectl get nodes

# Step 4. Deploy Katib components.
echo -e "\nDeploying Katib components\n"
kubectl apply -k "github.com/kubeflow/katib.git/manifests/v1beta1/installs/katib-standalone?ref=master"

# Wait until all Katib pods are running.
kubectl wait --for=condition=ready --timeout=${TIMEOUT} -l "katib.kubeflow.org/component in (controller,db-manager,mysql,ui)" -n kubeflow pod

echo -e "\nKatib has been deployed"
kubectl get pods -n kubeflow

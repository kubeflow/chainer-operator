#!/bin/bash

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This shell script is used to build a cluster and create a namespace from our
# argo workflow


set -xe
set -o pipefail

export CLUSTER_NAME="${CLUSTER_NAME}"
export ZONE="${GCP_ZONE}"
export PROJECT="${GCP_PROJECT}"
export K8S_NAMESPACE="${DEPLOY_NAMESPACE}"
export REGISTRY="${GCP_REGISTRY}"
KFCTL_DIR="${KFCTL_DIR}"
WORK_DIR=$(mktemp -d)
VERSION=$(git describe --tags --always --dirty)
source `dirname $0`/kfctl-util.sh
source `dirname $0`/gcloud-util.sh

gcloud::auth_activate
echo "Configuring kubectl"
gcloud --project ${PROJECT} container clusters get-credentials ${CLUSTER_NAME} \
    --zone ${ZONE}

cd ${WORK_DIR}

kfctl::init ${KFCTL_DIR} ${CLUSTER_NAME} ${PROJECT}

cd ${CLUSTER_NAME}
cat env.sh # for debugging
export CLIENT_ID=dummy
export CLIENT_SECRET=dummy
kfctl::generate ${KFCTL_DIR} all

cd $(source env.sh; echo ${KUBEFLOW_KS_DIR})

# kfctl.sh generate remove default env
ks env add default --namespace "${K8S_NAMESPACE}" 

echo "Disable spartakus"
ks param set spartakus reportUsage false

echo "Install the operator"
ks pkg install kubeflow/chainer-job
ks generate chainer-operator chainer-operator --image=${REGISTRY}/${REPO_NAME}:${VERSION}
ks apply default -c chainer-operator
TIMEOUT=30
until kubectl get pods -n ${K8S_NAMESPACE} | grep chainer-operator | grep 1/1 || [[ $TIMEOUT -eq 1 ]]; do
  kubectl get pods -n ${K8S_NAMESPACE}
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done

if [[ $TIMEOUT -eq 1 ]]; then
 exit 1
fi

echo "Run 'ChainerJob' test"
MNIST_TEST="chainer-mnist-test"
ks generate chainer-job-simple ${MNIST_TEST} \
  --image=everpeace/chainer:4.1.0 \
  --gpus=0 \
  --command=python3 \
  --args='/train_mnist.py,-e,2,-b,1000,-u,100,--noplot'
ks apply default -c ${MNIST_TEST}
TIMEOUT=30
until [[ $(kubectl -n ${K8S_NAMESPACE} get chj ${MNIST_TEST} -ojsonpath={.status.succeeded}) == 1 ]] || [[ $TIMEOUT -eq 1 ]] ; do
  kubectl -n ${K8S_NAMESPACE} get pods
  kubectl -n ${K8S_NAMESPACE} get chj ${MNIST_TEST}
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done
if [[ $TIMEOUT -eq 1 ]]; then
 exit 1
fi

MN_MNIST_TEST_IMAGE="chainermn-mnist-test"
ks generate chainer-job ${MN_MNIST_TEST_IMAGE} \
  --image=everpeace/chainermn:1.3.0 \
  --workers=1 \
  --workerSetName=ws \
  --gpus=0 \
  --command=python3 \
  --args='/train_mnist.py,-e,2,-b,1000,-u,100'
ks apply default -c ${MN_MNIST_TEST_IMAGE}
TIMEOUT=30
until [[ $(kubectl -n ${K8S_NAMESPACE} get chj ${MN_MNIST_TEST_IMAGE} -ojsonpath={.status.succeeded}) == 1 ]] || [[ $TIMEOUT -eq 1 ]] ; do
  kubectl -n ${K8S_NAMESPACE} get pods
  kubectl -n ${K8S_NAMESPACE} get chj ${MN_MNIST_TEST_IMAGE}
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done
if [[ $TIMEOUT -eq 1 ]]; then
 exit 1
fi

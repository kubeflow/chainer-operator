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


set -o errexit
set -o nounset
set -o pipefail

CLUSTER_NAME="${CLUSTER_NAME}"
ZONE="${GCP_ZONE}"
PROJECT="${GCP_PROJECT}"
K8S_NAMESPACE="${DEPLOY_NAMESPACE}"
KFCTL_DIR="${KFCTL_DIR}"
WORK_DIR=$(mktemp -d)
VERSION=$(git describe --tags --always --dirty)
source `dirname $0`/kfctl-util.sh

cd ${WORK_DIR}

kfctl::init ${KFCTL_DIR} ${CLUSTER_NAME} ${PROJECT}

cd ${CLUSTER_NAME}
cat env.sh # for debugging
kfctl::generate ${KFCTL_DIR} all

cd $(source env.sh; echo ${KUBEFLOW_KS_DIR})

echo "Install the operator"
ks pkg install kubeflow/chainer-job
ks generate chainer-operator chainer-operator --image=${REGISTRY}/${REPO_NAME}:${VERSION}
ks apply default -c chainer-operator
TIMEOUT=30
until kubectl get pods -n ${NAMESPACE} | grep chainer-operator | grep 1/1 || [[ $TIMEOUT -eq 1 ]]; do
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done

echo "Run 'ChainerJob' test"
MNIST_TEST="chainer-mnist-test"
ks generate chainer-job-simple ${MNIST_TEST} \
  --image=${REGISTRY}/${MNIST_TEST}:${VERSION} \
  --gpus=0 \
  --command=python3 \
  --args='/train_mnist.py,-e,2,-b,1000,-u,100,--noplot'
ks apply ${KF_ENV} -c ${MNIST_TEST}
TIMEOUT=30
until [[ $(kubectl -n ${NAMESPACE} get chj ${MNIST_TEST} -ojsonpath={.status.succeeded}) == 1 ]] ; do
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done


MN_MNIST_TEST_IMAGE="chainermn-mnist-test"
ks generate chainer-job ${MN_MNIST_TEST_IMAGE} \
  --image=${REGISTRY}/${MN_MNIST_TEST_IMAGE}:${VERSION} \
  --workers=1 \
  --workerSetName=ws \
  --gpus=0 \
  --command=python3 \
  --args='/train_mnist.py,-e,2,-b,1000,-u,100'
ks apply ${KF_ENV} -c ${MN_MNIST_TEST_IMAGE}
TIMEOUT=30
until [[ $(kubectl -n ${NAMESPACE} get chj ${MN_MNIST_TEST_IMAGE} -ojsonpath={.status.succeeded}) == 1 ]] ; do
  sleep 10
  TIMEOUT=$(( TIMEOUT - 1 ))
done

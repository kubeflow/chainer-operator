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
NAMESPACE="${DEPLOY_NAMESPACE}"
REGISTRY="${GCP_REGISTRY}"
VERSION=$(git describe --tags --always --dirty)
APP_NAME=test-app
KUBEFLOW_VERSION=master
KF_ENV=chainer

echo "Activating service-account"
gcloud auth activate-service-account --key-file=${GOOGLE_APPLICATION_CREDENTIALS}
echo "Configuring kubectl"
gcloud --project ${PROJECT} container clusters get-credentials ${CLUSTER_NAME} \
    --zone ${ZONE}

ACCOUNT=`gcloud config get-value account --quiet`
echo "Setting account ${ACCOUNT}"
kubectl create clusterrolebinding default-admin --clusterrole=cluster-admin --user=${ACCOUNT}

echo "Install ksonnet app in namespace ${NAMESPACE}"
ks init ${APP_NAME}
cd ${APP_NAME}
ks env add ${KF_ENV}
ks env set ${KF_ENV} --namespace ${NAMESPACE}
ks registry add kubeflow github.com/kubeflow/kubeflow/tree/${KUBEFLOW_VERSION}/kubeflow

echo "Install the operator"
ks pkg install kubeflow/chainer-job@${KUBEFLOW_VERSION}
ks generate chainer-operator chainer-operator --image=${REGISTRY}/${REPO_NAME}:${VERSION}
ks apply ${KF_ENV} -c chainer-operator

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

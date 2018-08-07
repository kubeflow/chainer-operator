#!/bin/bash

# Copyright 2018 The Kubeflow Authors.
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

# This shell script is used to build an image from our argo workflow

set -o errexit
set -o nounset
set -o pipefail

export PATH=${GOPATH}/bin:/usr/local/go/bin:${PATH}
REGISTRY="${GCP_REGISTRY}"
PROJECT="${GCP_PROJECT}"
VERSION=$(git describe --tags --always --dirty)

echo "Activating service-account"
gcloud auth activate-service-account --key-file=${GOOGLE_APPLICATION_CREDENTIALS}
echo "building chainer operator image in gcloud"
gcloud version
# gcloud components update -q
# build chainer operator image
gcloud container builds submit . --tag=${REGISTRY}/${REPO_NAME}:${VERSION} --project=${PROJECT}

# build a mnist testing image for our smoke test
echo "building chainer's mnist testing image in gcloud"
MNIST_TEST_IMAGE="chainer-mnist-test"
gcloud container builds submit ./examples/docker/chainer/ --tag=${REGISTRY}/${MNIST_TEST_IMAGE}:${VERSION} --project=${PROJECT}

echo "building chainermn's mnist testing image in gcloud"
MN_MNIST_TEST_IMAGE="chainermn-mnist-test"
gcloud container builds submit ./examples/docker/chainermn/ --tag=${REGISTRY}/${MN_MNIST_TEST_IMAGE}:${VERSION} --project=${PROJECT}
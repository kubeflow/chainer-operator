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

CLUSTER_NAME="${CLUSTER_NAME}"
ZONE="${GCP_ZONE}"
PROJECT="${GCP_PROJECT}"
K8S_NAMESPACE="${DEPLOY_NAMESPACE}"
KFCTL_DIR=${KFCTL_DIR}
WORK_DIR=$(mktemp -d)
source `dirname $0`/kfctl-util.sh
source `dirname $0`/gcloud-util.sh

gcloud::auth_activate

cd ${WORK_DIR}

kfctl::init ${KFCTL_DIR} ${CLUSTER_NAME} ${PROJECT}

cd ${CLUSTER_NAME}
cat env.sh # for debugging

export CLIENT_ID=dummy
export CLIENT_SECRET=dummy
kfctl::generate ${KFCTL_DIR} platform
kfctl::apply ${KFCTL_DIR} platform

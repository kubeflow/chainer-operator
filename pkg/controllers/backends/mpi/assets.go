// Copyright 2018 The Kubeflow Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mpi

import (
	"strings"

	apisv1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
)

var (
	kubexecSh = strings.Replace(strings.TrimSpace(`#! /bin/sh
pod=$1
shift
if [ "$pod" = $(hostname) ]; then
	$@
else
	${KUBECTL_DIR:-/kubeflow/chainer-operator/kube}/kubectl exec -i $pod -c %%JOB_CONTAINER_NAME%% -- $@
fi
`), "%%JOB_CONTAINER_NAME%%", apisv1alpha1.DefaultContainerName, -1)

	kubectlDownloadSh = strings.TrimSpace(`#! /bin/sh
set -ex

TARGET=${1:-/kubeflow/chainer-operator/kube}
MAX_TRY=${2:-10}
SLEEP_SECS=${3:-5}
TRIED=0
COMMAND_STATUS=1

until [ $COMMAND_STATUS -eq 0 ] || [ $TRIED -eq $MAX_TRY ]; do
	curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && chmod +x kubectl && mv kubectl $TARGET/kubectl
	COMMAND_STATUS=$?
	sleep $SLEEP_SECS
	TRIED=$(expr $TRIED + 1)
done
`)

	genHostfileSh = `#! /bin/sh
set -xev

KUBECTL_DIR=${KUBECTL_DIR:-/kubeflow/chainer-operator/kube}
ASSETS_DIR=${ASSETS_DIR:-/kubeflow/chainer-operator/assets}
KUBECTL=${KUBECTL_DIR}/kubectl

TARGET=${1}
MASTER_POD_NAME=${2}
MASTER_SLOTS=${3:-slots=1}
STATEFUL_SETS_AND_SLOTS_FILE=${4:-${ASSETS_DIR}/statefulsets_and_slots}
MAX_TRY=${5:-100}
SLEEP_SECS=${6:-5}

trap "rm -f ${TARGET}_new" EXIT TERM INT KILL

cluster_size=1
for ss in $(cat ${STATEFUL_SETS_AND_SLOTS_FILE} | cut -d' ' -f 1); do
	replicas=$($KUBECTL get statefulsets ${ss} -o=jsonpath='{.status.replicas}')
	cluster_size=$(expr $cluster_size + $replicas)
done

tried=0
until [ "$(wc -l < ${TARGET}_new)" -eq $cluster_size ]; do
	rm -f ${TARGET}_new

	cat ${STATEFUL_SETS_AND_SLOTS_FILE} | while read ss_s; do
		ss=$(echo "${ss_s}" | cut -d' ' -f 1)
		slots=$(echo "${ss_s}" | cut -d' ' -f 2)
		replicas=$($KUBECTL get statefulsets "${ss}" -o=jsonpath='{.status.replicas}')

		for i in $(seq 0 ${replicas} | sed -e '$,$d'); do
			phase=$($KUBECTL get pod "${ss}-${i}" -o=jsonpath='{.status.phase}')
			if [ "$phase" = "Running" ]; then
				echo "${ss}-${i} ${slots}" >> ${TARGET}_new
			fi
		done
	done
	
	echo "${MASTER_POD_NAME} ${MASTER_SLOTS}" >> ${TARGET}_new

	tried=$(expr $tried + 1)
	if [ $tried -ge $MAX_TRY ]; then
		break
	fi
	sleep $SLEEP_SECS
done

if [ -e ${TARGET}_new ]; then
	mv ${TARGET}_new ${TARGET}
fi	
`
)

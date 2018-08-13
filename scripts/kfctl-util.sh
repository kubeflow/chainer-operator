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

function kfctl::_simple_exec(){
    local kfctl_dir=$1
    local command=$2
    local what=$3

    echo "executing kfctl ${command} ${what}"
    ${kfctl_dir}/scripts/kfctl.sh ${command} ${what}
}

function kfctl::init(){
    local kfctl_dir=$1
    local deploy_name=$2
    local gcp_project=$3

    echo "Initializing kfctl"
    ${kfctl_dir}/scripts/kfctl.sh init ${deploy_name} \
      --platform gcp \
      --project ${gcp_project}
}

function kfctl::generate(){
    local kfctl_dir=$1
    local what=$2
    kfctl::_simple_exec ${kfctl_dir} generate ${what}
}

function kfctl::apply(){
    local kfctl_dir=$1
    local what=$2
    kfctl::_simple_exec ${kfctl_dir} apply ${what}
}

function kfctl::delete(){
    local kfctl_dir=$1
    local what=$2
    kfctl::_simple_exec ${kfctl_dir} delete ${what}
}

// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/runtime"
)

// Int32 is a helper routine that allocates a new int32 value
// to store v and returns a pointer to it.
func Int32(v int32) *int32 {
	return &v
}

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

func setWorkerSetSpecReplicas(spec *WorkerSetSpec, name string) {
	if spec.Replicas == nil {
		spec.Replicas = Int32(1)
		glog.V(4).Infof("setting spec.WorkerSets[%s].Replicas to 1", name)
	}
}

func setWorkerSetSpecContainerName(spec *WorkerSetSpec, name string) {
	containers := spec.Template.Spec.Containers
	if len(containers) == 1 && containers[0].Name == "" {
		containers[0].Name = DefaultContainerName
		glog.V(4).Infof("setting spec.WorkerSets[%s].Template.Spec.Containers[0].Name to %s", name, DefaultContainerName)
	}
}

func setWorkerSetMPISlots(spec *WorkerSetSpec, name string) {
	if spec.MPIConfig == nil || spec.MPIConfig.Slots == nil {
		for _, container := range spec.Template.Spec.Containers {
			if container.Name == DefaultContainerName {
				slots := Int32(DefaultSlots)
				if q, ok := container.Resources.Limits["nvidia.com/gpu"]; ok {
					slots = Int32(int32(q.Value()))
				}
				spec.MPIConfig = defaultMPIConfig(slots)
				glog.V(4).Infof("setting spec.WorkerSets[%+v].MPIConfig.Slots to %+v", name, *spec.MPIConfig.Slots)
				return
			}
		}
		spec.MPIConfig = defaultMPIConfig(Int32(DefaultSlots))
	}
}

func setMPIMasterSpecContainerName(spec *MasterSpec) {
	containers := spec.Template.Spec.Containers
	if len(containers) == 1 && containers[0].Name == "" {
		containers[0].Name = DefaultContainerName
		glog.V(4).Infof("setting spec.MasterSpec.Template.Spec.Containers[0].Name to %s", DefaultContainerName)
	}
}

func defaultMPIConfig(slots *int32) *MPIConfig {
	return &MPIConfig{
		Slots: slots,
	}
}

func setMasterSpecMPISlots(spec *MasterSpec) {
	if spec.MPIConfig == nil || spec.MPIConfig.Slots == nil {
		for _, container := range spec.Template.Spec.Containers {
			if container.Name == DefaultContainerName {
				slots := Int32(DefaultSlots)
				if q, ok := container.Resources.Limits["nvidia.com/gpu"]; ok {
					slots = Int32(int32(q.Value()))
				}
				spec.MPIConfig = defaultMPIConfig(slots)
				glog.V(4).Infof("setting spec.Master.MPIConfig.Slots to %+v", *spec.MPIConfig.Slots)
				return
			}
		}
		spec.MPIConfig = defaultMPIConfig(Int32(DefaultSlots))
	}
}

func setMPIMasterRetryPolicy(spec *MasterSpec) {
	if spec.Template.Spec.RestartPolicy == "" {
		spec.Template.Spec.RestartPolicy = DefaultRestartPolicy
		glog.V(4).Infof("setting spec.MPISpec.MasterSpec.Spec.RestartPolicy to %s", DefaultRestartPolicy)
	}
}

func setDefaultBackend(chainerjob *ChainerJob) {
	if chainerjob.Spec.Backend == "" {
		chainerjob.Spec.Backend = BackendTypeMPI
	}
}

// SetDefaults_ChainerJob sets any unspecified values to defaults.
func SetDefaults_ChainerJob(chainerjob *ChainerJob) {
	glog.V(4).Infof("start setting default to %s", chainerjob.Name)
	setMPIMasterSpecContainerName(&chainerjob.Spec.Master)
	setMPIMasterRetryPolicy(&chainerjob.Spec.Master)

	if IsDistributed(chainerjob) {
		setDefaultBackend(chainerjob)
		for name, workerSetSpec := range chainerjob.Spec.WorkerSets {
			setWorkerSetSpecReplicas(workerSetSpec, name)
			setWorkerSetSpecContainerName(workerSetSpec, name)
		}
		switch chainerjob.Spec.Backend {
		case BackendTypeMPI:
			setMasterSpecMPISlots(&chainerjob.Spec.Master)
			for name, workerSetSpec := range chainerjob.Spec.WorkerSets {
				setWorkerSetMPISlots(workerSetSpec, name)
			}
		default:
			// nop
		}
	}
	glog.V(4).Infof("finish setting default to %s", chainerjob.Name)
}

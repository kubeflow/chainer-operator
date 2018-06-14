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
	"bytes"
	"fmt"
	"strconv"

	"github.com/golang/glog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"k8s.io/client-go/kubernetes"
	appslister "k8s.io/client-go/listers/apps/v1"
	batchlister "k8s.io/client-go/listers/batch/v1"
	corelister "k8s.io/client-go/listers/core/v1"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/record"

	apisv1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
	clientset "github.com/kubeflow/chainer-operator/pkg/client/clientset/versioned"

	"github.com/kubeflow/chainer-operator/pkg/controllers/backends"
)

const (
	assetsSuffix = "-assets"

	kubectlDownloaderContainerName = "chainer-operator-kubectl-downloader"
	hostfileGeneratorContainerName = "chainer-operator-hostfile-generator"

	kubectlDownloadScriptName    = "download_kubectl.sh"
	kubexecScriptName            = "kubexeb.sh"
	genHostfileScriptName        = "gen_hostfile.sh"
	statefulSetsAndSlotsFileName = "statefulsets_and_slots"

	volumeNameBase     = "chainer-operator-volume"
	assetsVolumeName   = volumeNameBase + "-assets"
	kubectlVolumeName  = volumeNameBase + "-kubectl"
	hostfileVolumeName = volumeNameBase + "-generated"
	mountPathBase      = "/kubeflow/chainer-operator"
	assetsMountPath    = mountPathBase + "/assets"
	kubectlMountPath   = mountPathBase + "/kube"
	hostfileMountPath  = mountPathBase + "/generated"
	hostfileName       = "hostfile"

	kubectlDownloaderContainerImage = "tutum/curl"
	hostfileGeneratorContainerImage = "alpine"
	kubectlDirEnv                   = "KUBECTL_DIR"

	podManagementPolicy = appsv1.ParallelPodManagement
)

// Backend is responsible for syncing ChainerJob with backend:mpi
type Backend struct {
	kubeClient           kubernetes.Interface
	kubeflowClient       clientset.Interface
	serviceAccountLister corelister.ServiceAccountLister
	roleLister           rbaclister.RoleLister
	roleBindingLister    rbaclister.RoleBindingLister
	configMapLister      corelister.ConfigMapLister
	jobLister            batchlister.JobLister
	statefulSetLister    appslister.StatefulSetLister

	recorder record.EventRecorder
}

// NewBackend is constructor for Backend
func NewBackend(
	kubeClient kubernetes.Interface,
	kubeflowClient clientset.Interface,
	serviceAccountLister corelister.ServiceAccountLister,
	roleLister rbaclister.RoleLister,
	roleBindingLister rbaclister.RoleBindingLister,
	configMapLister corelister.ConfigMapLister,
	jobLister batchlister.JobLister,
	statefulSetLister appslister.StatefulSetLister,
	recorder record.EventRecorder,
) backends.Backend {
	return &Backend{
		kubeClient:           kubeClient,
		kubeflowClient:       kubeflowClient,
		serviceAccountLister: serviceAccountLister,
		roleLister:           roleLister,
		roleBindingLister:    roleBindingLister,
		configMapLister:      configMapLister,
		jobLister:            jobLister,
		statefulSetLister:    statefulSetLister,
		recorder:             recorder,
	}
}

// SyncChainerJob is main function to sync ChainerJob with "backend: mpi"
func (b *Backend) SyncChainerJob(chjob *apisv1alpha1.ChainerJob) error {
	if !apisv1alpha1.IsDistributed(chjob) || chjob.Spec.Backend != apisv1alpha1.BackendTypeMPI {
		return fmt.Errorf("not syncing %s because it is not a mpi backended distributed job", chjob.Name)
	}

	sa, err := b.syncServiceAccount(chjob)
	if sa == nil || err != nil {
		return err
	}
	glog.V(4).Infof("syncing %s: serviceaccount %s synced.", chjob.Name, sa.Name)

	role, err := b.syncRole(chjob)
	if role == nil || err != nil {
		return err
	}
	glog.V(4).Infof("syncing %s: role %s synced.", chjob.Name, role.Name)

	rolebinding, err := b.syncRoleBinding(chjob)
	if rolebinding == nil || err != nil {
		return err
	}
	glog.V(4).Infof("syncing %s: rolebinding %s synced.", chjob.Name, rolebinding.Name)

	if apisv1alpha1.IsDistributed(chjob) {
		cm, err := b.syncConfigMap(chjob)
		if cm == nil || err != nil {
			return err
		}
		glog.V(4).Infof("syncing %s: configmap %s synced.", chjob.Name, cm.Name)
	}

	master, err := b.syncMaster(chjob)
	if master == nil || err != nil {
		return err
	}
	glog.V(4).Infof("syncing %s: job %s synced.", chjob.Name, master.Name)

	if apisv1alpha1.IsDistributed(chjob) {
		workerSets, err := b.syncWorkerSets(chjob, isJobDone(master))
		if len(workerSets) == 0 || err != nil {
			return err
		}
		for _, ws := range workerSets {
			glog.V(4).Infof("syncing %s: statefulset %s synced.", chjob.Name, ws.Name)
		}
	}

	err = backends.UpdateChainerJobStatus(chjob, &master.Status, b.kubeflowClient)
	if err != nil {
		return err
	}
	glog.V(4).Infof("syncing %s: updated status.", chjob.Name)

	return nil
}

func (b *Backend) syncServiceAccount(chjob *apisv1alpha1.ChainerJob) (*corev1.ServiceAccount, error) {
	return backends.CreateServiceAccountIfNotExist(
		chjob,
		b.kubeClient,
		b.serviceAccountLister,
		b.recorder,
		backends.NewServiceAccount,
	)
}

func (b *Backend) syncRole(chjob *apisv1alpha1.ChainerJob) (*rbacv1.Role, error) {
	return backends.CreateOrUpdateRole(
		chjob,
		b.kubeClient,
		b.roleLister,
		b.recorder,
		newRole,
	)
}

func newRole(chjob *apisv1alpha1.ChainerJob) *rbacv1.Role {
	podNames := make([]string, 0)
	ssNames := make([]string, 0)
	for name, workerSpec := range chjob.Spec.WorkerSets {
		ssName := fmt.Sprintf("%s%s-%s", chjob.Name, backends.WorkerSetSuffix, name)
		ssNames = append(ssNames, ssName)
		for i := 0; i < int(*workerSpec.Replicas); i++ {
			podNames = append(podNames, fmt.Sprintf("%s-%d", ssName, i))
		}
	}
	rules := []rbacv1.PolicyRule{
		rbacv1.PolicyRule{
			APIGroups:     []string{"apps"},
			Verbs:         []string{"get"},
			Resources:     []string{"statefulsets"},
			ResourceNames: ssNames,
		},
		rbacv1.PolicyRule{
			APIGroups:     []string{""},
			Verbs:         []string{"get", "list"},
			Resources:     []string{"pods"},
			ResourceNames: podNames,
		},
		rbacv1.PolicyRule{
			APIGroups:     []string{""},
			Verbs:         []string{"create"},
			Resources:     []string{"pods/exec"},
			ResourceNames: podNames,
		},
	}
	role := backends.NewRole(chjob)
	role.Rules = rules
	return role
}

func (b *Backend) syncRoleBinding(chjob *apisv1alpha1.ChainerJob) (*rbacv1.RoleBinding, error) {
	return backends.CreateOrUpdateRoleBinding(
		chjob,
		b.kubeClient,
		b.roleBindingLister,
		b.recorder,
		backends.NewRoleBindings,
	)
}

func (b *Backend) syncConfigMap(chjob *apisv1alpha1.ChainerJob) (*corev1.ConfigMap, error) {
	return backends.CreateOrUpdateConfigMap(
		chjob,
		b.kubeClient,
		b.configMapLister,
		b.recorder,
		newConfigMap,
	)
}

func (b *Backend) syncMaster(chjob *apisv1alpha1.ChainerJob) (*batchv1.Job, error) {
	return backends.CreateJobIfNotExist(
		chjob,
		b.kubeClient,
		b.jobLister,
		b.recorder,
		newMasterJob,
	)
}

func (b *Backend) syncWorkerSets(chjob *apisv1alpha1.ChainerJob, done bool) ([]*appsv1.StatefulSet, error) {
	sss := make([]*appsv1.StatefulSet, 0)

	for name := range chjob.Spec.WorkerSets {
		ss, err := b.syncWorkerSet(chjob, done, name)
		if err != nil {
			return sss, err
		}
		sss = append(sss, ss)
	}

	return sss, nil
}

func (b *Backend) syncWorkerSet(chjob *apisv1alpha1.ChainerJob, done bool, name string) (*appsv1.StatefulSet, error) {
	return backends.CreateOrUpdateStatefulSet(
		chjob,
		b.kubeClient,
		b.statefulSetLister,
		b.recorder,
		func(chj *apisv1alpha1.ChainerJob) *appsv1.StatefulSet {
			return newWorkerSet(chj, done, name)
		},
	)
}

func newConfigMap(chjob *apisv1alpha1.ChainerJob) *corev1.ConfigMap {
	var ssAndSlots bytes.Buffer
	for name, spec := range chjob.Spec.WorkerSets {
		ssAndSlots.WriteString(fmt.Sprintf("%s%s-%s slots=%d\n", chjob.Name, backends.WorkerSetSuffix, name, *spec.MPIConfig.Slots))
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chjob.Name + assetsSuffix,
			Namespace: chjob.Namespace,
			Labels: map[string]string{
				backends.JobLabelKey: chjob.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(chjob, apisv1alpha1.SchemeGroupVersionKind),
			},
		},
		Data: map[string]string{
			kubectlDownloadScriptName:    kubectlDownloadSh,
			kubexecScriptName:            kubexecSh,
			genHostfileScriptName:        genHostfileSh,
			statefulSetsAndSlotsFileName: ssAndSlots.String(),
		},
	}
}

func newManagedMPIVolumes(chjob *apisv1alpha1.ChainerJob) []corev1.Volume {
	scriptMode := int32(0555)
	return []corev1.Volume{
		corev1.Volume{
			Name: hostfileVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: kubectlVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: assetsVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: chjob.Name + assetsSuffix,
					},
					Items: []corev1.KeyToPath{
						corev1.KeyToPath{
							Key:  kubectlDownloadScriptName,
							Path: kubectlDownloadScriptName,
							Mode: &scriptMode,
						},
						corev1.KeyToPath{
							Key:  kubexecScriptName,
							Path: kubexecScriptName,
							Mode: &scriptMode,
						},
						corev1.KeyToPath{
							Key:  genHostfileScriptName,
							Path: genHostfileScriptName,
							Mode: &scriptMode,
						},
						corev1.KeyToPath{
							Key:  statefulSetsAndSlotsFileName,
							Path: statefulSetsAndSlotsFileName,
							Mode: &scriptMode,
						},
					},
				},
			},
		},
	}
}

func newManagedMPIVolumeMounts(chjob *apisv1alpha1.ChainerJob) []corev1.VolumeMount {
	return []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      hostfileVolumeName,
			MountPath: hostfileMountPath,
		},
		corev1.VolumeMount{
			Name:      kubectlVolumeName,
			MountPath: kubectlMountPath,
		},
		corev1.VolumeMount{
			Name:      assetsVolumeName,
			MountPath: assetsMountPath,
		},
	}
}

func newHostfileGeneratorContainer(volumeMounts []corev1.VolumeMount, masterSlots *int32) []corev1.Container {
	return []corev1.Container{
		corev1.Container{
			Name:         hostfileGeneratorContainerName,
			Image:        "alpine",
			VolumeMounts: volumeMounts,
			Env: []corev1.EnvVar{
				corev1.EnvVar{
					Name: "POD_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
			},
			Command: []string{
				assetsMountPath + "/" + genHostfileScriptName,
			},
			Args: []string{
				hostfileMountPath + "/" + hostfileName,
				"$(POD_NAME)",
				"slots=" + strconv.Itoa(int(*masterSlots)),
				assetsMountPath + "/" + statefulSetsAndSlotsFileName,
			},
		},
	}
}

func newKubectlDownloaderContainer(volumeMounts []corev1.VolumeMount) []corev1.Container {
	return []corev1.Container{
		corev1.Container{
			Name:  kubectlDownloaderContainerName,
			Image: kubectlDownloaderContainerImage,
			Command: []string{
				assetsMountPath + "/" + kubectlDownloadScriptName,
			},
			Args:         []string{kubectlMountPath},
			VolumeMounts: volumeMounts,
		},
	}
}

func newManagedMPIEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		corev1.EnvVar{
			Name:  kubectlDirEnv,
			Value: kubectlMountPath,
		},
		corev1.EnvVar{
			Name:  "OMPI_MCA_plm_rsh_agent",
			Value: assetsMountPath + "/" + kubexecScriptName,
		},
		corev1.EnvVar{
			Name:  "OMPI_MCA_orte_keep_fqdn_hostnames",
			Value: "t",
		},
		corev1.EnvVar{
			Name:  "OMPI_MCA_orte_default_hostfile",
			Value: hostfileMountPath + "/" + hostfileName,
		},
		corev1.EnvVar{
			Name:  "OMPI_MCA_btl_tcp_if_exclude",
			Value: "lo,docker0",
		},
	}
}

func newMasterJob(chjob *apisv1alpha1.ChainerJob) *batchv1.Job {
	// generate base job
	job := backends.NewMasterJob(chjob)

	// decorating base job
	// manged parts to be injected.
	managedVolumes := newManagedMPIVolumes(chjob)
	managedVolumeMounts := newManagedMPIVolumeMounts(chjob)
	managedInitContainers := append(
		newKubectlDownloaderContainer(managedVolumeMounts),
		newHostfileGeneratorContainer(managedVolumeMounts, chjob.Spec.Master.MPIConfig.Slots)...)
	managedEnv := newManagedMPIEnvVars()

	// inject aboves to user defined podTemplate
	podTemplate := job.Spec.Template.DeepCopy()
	podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, managedVolumes...)
	modifiedInitContainers := make([]corev1.Container, len(podTemplate.Spec.InitContainers))
	for i, initContainer := range podTemplate.Spec.InitContainers {
		modifiedInitContainer := initContainer.DeepCopy()
		modifiedInitContainer.VolumeMounts = append(modifiedInitContainer.VolumeMounts, managedVolumeMounts...)
		modifiedInitContainers[i] = *modifiedInitContainer
	}
	podTemplate.Spec.InitContainers = append(modifiedInitContainers, managedInitContainers...)
	modifiedContainers := make([]corev1.Container, len(podTemplate.Spec.Containers))
	for i, container := range podTemplate.Spec.Containers {
		modifiedContainer := container.DeepCopy()
		modifiedContainer.Env = append(modifiedContainer.Env, managedEnv...)
		modifiedContainer.VolumeMounts = append(modifiedContainer.VolumeMounts, managedVolumeMounts...)
		modifiedContainers[i] = *modifiedContainer
	}
	podTemplate.Spec.Containers = modifiedContainers

	return &batchv1.Job{
		ObjectMeta: job.ObjectMeta,
		Spec: batchv1.JobSpec{
			Completions:           job.Spec.Completions,
			Parallelism:           job.Spec.Parallelism,
			BackoffLimit:          job.Spec.BackoffLimit,
			ActiveDeadlineSeconds: job.Spec.ActiveDeadlineSeconds,
			Template:              *podTemplate,
		},
	}
}

func newWorkerSet(chjob *apisv1alpha1.ChainerJob, done bool, name string) *appsv1.StatefulSet {
	workerSet := backends.NewWorkerSet(chjob, done, name)

	// manged parts to be injected.
	managedVolumes := newManagedMPIVolumes(chjob)
	managedVolumeMounts := newManagedMPIVolumeMounts(chjob)
	managedInitContainers := newKubectlDownloaderContainer(managedVolumeMounts)
	managedEnv := newManagedMPIEnvVars()

	// inject aboves to user defined template
	podTemplate := workerSet.Spec.Template.DeepCopy()
	podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, managedVolumes...)
	modifiedInitContainers := make([]corev1.Container, len(podTemplate.Spec.InitContainers))
	for i, initContainer := range podTemplate.Spec.InitContainers {
		modifiedInitContainer := initContainer.DeepCopy()
		modifiedInitContainer.VolumeMounts = append(modifiedInitContainer.VolumeMounts, managedVolumeMounts...)
		modifiedInitContainers[i] = *modifiedInitContainer
	}
	podTemplate.Spec.InitContainers = append(modifiedInitContainers, managedInitContainers...)
	modifiedContainers := make([]corev1.Container, len(podTemplate.Spec.Containers))
	for i, container := range podTemplate.Spec.Containers {
		modifiedContainer := container.DeepCopy()
		modifiedContainer.Env = append(modifiedContainer.Env, managedEnv...)
		modifiedContainer.VolumeMounts = append(modifiedContainer.VolumeMounts, managedVolumeMounts...)
		modifiedContainers[i] = *modifiedContainer
	}
	podTemplate.Spec.Containers = modifiedContainers

	return &appsv1.StatefulSet{
		ObjectMeta: workerSet.ObjectMeta,
		Spec: appsv1.StatefulSetSpec{
			PodManagementPolicy: workerSet.Spec.PodManagementPolicy,
			Replicas:            workerSet.Spec.Replicas,
			Selector:            workerSet.Spec.Selector,
			ServiceName:         workerSet.Spec.ServiceName,
			Template:            *podTemplate,
		},
	}
}

func isJobDone(job *batchv1.Job) bool {
	succeeded := false
	failed := false
	if job != nil {
		for _, cond := range job.Status.Conditions {
			if cond.Type == "Complete" {
				succeeded = (cond.Status == "True")
			}
		}

		for _, cond := range job.Status.Conditions {
			if cond.Type == "Failed" {
				failed = (cond.Status == "True")
			}
		}
	}
	return job != nil && (succeeded || failed)
}

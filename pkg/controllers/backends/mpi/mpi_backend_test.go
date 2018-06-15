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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/kubeflow/chainer-operator/pkg/controllers/backends"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apisv1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

var (
	managedVolumeMounts = []corev1.VolumeMount{
		simpleVolumeMount(hostfileVolumeName, hostfileMountPath),
		simpleVolumeMount(kubectlVolumeName, kubectlMountPath),
		simpleVolumeMount(assetsVolumeName, assetsMountPath),
	}

	kubectlDownloaderContainer = corev1.Container{
		Name:         kubectlDownloaderContainerName,
		Image:        kubectlDownloaderContainerImage,
		VolumeMounts: managedVolumeMounts,
		Command:      []string{assetsMountPath + "/" + kubectlDownloadScriptName},
		Args:         []string{kubectlMountPath},
	}

	managedEnvVars = []corev1.EnvVar{
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
)

func assetsVolume(cmName string, scriptMode int32) corev1.Volume {
	return corev1.Volume{
		Name: assetsVolumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmName,
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
	}
}

func emptyDirVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func simpleEnvVar(name string, value string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  name,
		Value: value,
	}
}

func simpleVolumeMount(name string, mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}
}

func simpleContainer(name string, image string) corev1.Container {
	return corev1.Container{
		Name:  name,
		Image: image,
	}
}

func simpleContainer2(name string, image string, envVars []corev1.EnvVar, volumeMounts []corev1.VolumeMount) corev1.Container {
	return corev1.Container{
		Name:         name,
		Image:        image,
		Env:          envVars,
		VolumeMounts: volumeMounts,
	}
}

func ownerReference(name string) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		{
			APIVersion:         apisv1alpha1.GroupName + "/" + apisv1alpha1.GroupVersion,
			Kind:               apisv1alpha1.Kind,
			Name:               name,
			UID:                "",
			Controller:         Bool(true),
			BlockOwnerDeletion: Bool(true),
		},
	}
}

func hostfileGeneratorContainer(slots *int32) corev1.Container {
	return corev1.Container{
		Name:         hostfileGeneratorContainerName,
		Image:        hostfileGeneratorContainerImage,
		VolumeMounts: managedVolumeMounts,
		Command:      []string{assetsMountPath + "/" + genHostfileScriptName},
		Args: []string{
			hostfileMountPath + "/" + hostfileName,
			"$(POD_NAME)",
			fmt.Sprintf("slots=%d", *slots),
			assetsMountPath + "/" + statefulSetsAndSlotsFileName,
		},
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
	}
}

func Int64(v int64) *int64 {
	return &v
}

func Bool(b bool) *bool {
	return &b
}

func TestNewMaster(t *testing.T) {
	// single and multiple
	type testCase struct {
		name     string
		in       *apisv1alpha1.ChainerJob
		expected *batchv1.Job
	}
	testCases := []testCase{
		{
			name: "multi-pods,multi-[init]containers,customVolumes",
			in: &apisv1alpha1.ChainerJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "chj",
					Namespace: "ch",
					Labels: map[string]string{
						"custom": "value",
					},
				},
				Spec: apisv1alpha1.ChainerJobSpec{
					Backend: apisv1alpha1.BackendTypeMPI,
					Master: apisv1alpha1.MasterSpec{
						ActiveDeadlineSeconds: Int64(1000),
						BackoffLimit:          apisv1alpha1.Int32(10),
						MPIConfig: &apisv1alpha1.MPIConfig{
							Slots: apisv1alpha1.Int32(3),
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"custom2": "value2",
								},
							},
							Spec: corev1.PodSpec{
								RestartPolicy: "Never",
								Volumes: []corev1.Volume{
									emptyDirVolume("customv"),
								},
								InitContainers: []corev1.Container{
									simpleContainer2("init1", "dummy",
										[]corev1.EnvVar{simpleEnvVar("ENV", "VAL")},
										[]corev1.VolumeMount{simpleVolumeMount("customv", "/customv")},
									),
								},
								Containers: []corev1.Container{
									simpleContainer2("chainer", "dummy",
										[]corev1.EnvVar{simpleEnvVar("ENV", "VAL")},
										[]corev1.VolumeMount{simpleVolumeMount("customv", "/customv")},
									),
									simpleContainer("helper", "dummy"),
								},
							},
						},
					},
					WorkerSets: map[string]*apisv1alpha1.WorkerSetSpec{
						"ws0": &apisv1alpha1.WorkerSetSpec{
							Replicas: apisv1alpha1.Int32(1),
							MPIConfig: &apisv1alpha1.MPIConfig{
								Slots: apisv1alpha1.Int32(1),
							},
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: "dummy",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "chj" + backends.MasterSuffix,
					Namespace: "ch",
					Labels: map[string]string{
						"custom":              "value",
						backends.JobLabelKey:  "chj",
						backends.RoleLabelKey: backends.RoleMaster,
					},
					OwnerReferences: ownerReference("chj"),
				},
				Spec: batchv1.JobSpec{
					Parallelism:           apisv1alpha1.Int32(1),
					Completions:           apisv1alpha1.Int32(1),
					ActiveDeadlineSeconds: Int64(1000),
					BackoffLimit:          apisv1alpha1.Int32(10),
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"custom2":             "value2",
								backends.JobLabelKey:  "chj",
								backends.RoleLabelKey: backends.RoleMaster,
							},
						},
						Spec: corev1.PodSpec{
							ServiceAccountName: "chj" + backends.ServiceAccountSuffix,
							RestartPolicy:      "Never",
							Volumes: []corev1.Volume{
								emptyDirVolume("customv"),
								emptyDirVolume(hostfileVolumeName),
								emptyDirVolume(kubectlVolumeName),
								assetsVolume("chj"+assetsSuffix, int32(0555)),
							},
							InitContainers: []corev1.Container{
								simpleContainer2("init1", "dummy",
									[]corev1.EnvVar{simpleEnvVar("ENV", "VAL")},
									append(
										[]corev1.VolumeMount{simpleVolumeMount("customv", "/customv")}, managedVolumeMounts...,
									),
								),
								kubectlDownloaderContainer,
								hostfileGeneratorContainer(apisv1alpha1.Int32(3)),
							},
							Containers: []corev1.Container{
								simpleContainer2("chainer", "dummy",
									append(
										[]corev1.EnvVar{simpleEnvVar("ENV", "VAL")},
										managedEnvVars...,
									),
									append(
										[]corev1.VolumeMount{simpleVolumeMount("customv", "/customv")}, managedVolumeMounts...,
									),
								),
								simpleContainer2("helper", "dummy", managedEnvVars, managedVolumeMounts),
							},
						},
					},
				},
			},
		},
	}

	for _, c := range testCases {
		actualJob := newMasterJob(c.in)
		actualJSON, errActual := json.MarshalIndent(actualJob, "", "  ")
		actual := ""
		if errActual != nil {
			t.Errorf("Couldn't pretty format %v, error: %v", actualJob, errActual)
			actual = fmt.Sprintf("%+v", actualJob)
		} else {
			actual = string(actualJSON)
		}

		expectedJSON, errExpected := json.MarshalIndent(c.expected, "", "  ")
		expected := ""
		if errExpected != nil {
			t.Errorf("Couldn't pretty format %v, error: %v", c.expected, errExpected)
			expected = fmt.Sprintf("%v", c.expected)
		} else {
			expected = string(expectedJSON)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("TestCase: %v\nWant:%+v\nGot:%+v", c.name, expected, actual)
		}
	}
}

func TestNewWorkerSet(t *testing.T) {
	type testCase struct {
		name     string
		in       *apisv1alpha1.ChainerJob
		wsName   string
		done     bool
		expected *appsv1.StatefulSet
	}

	inputJob := &apisv1alpha1.ChainerJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "chj",
			Namespace: "ch",
			Labels: map[string]string{
				"custom": "value",
			},
		},
		Spec: apisv1alpha1.ChainerJobSpec{
			Backend: apisv1alpha1.BackendTypeMPI,
			Master: apisv1alpha1.MasterSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							simpleContainer("chainer", "dummy"),
						},
					},
				},
			},
			WorkerSets: map[string]*apisv1alpha1.WorkerSetSpec{
				"ws0": &apisv1alpha1.WorkerSetSpec{
					Replicas: apisv1alpha1.Int32(2),
					MPIConfig: &apisv1alpha1.MPIConfig{
						Slots: apisv1alpha1.Int32(3),
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"custom2": "value2",
							},
						},
						Spec: corev1.PodSpec{
							Volumes: []corev1.Volume{
								emptyDirVolume("customv"),
							},
							InitContainers: []corev1.Container{
								simpleContainer2("init1", "dummy",
									[]corev1.EnvVar{simpleEnvVar("ENV", "VAL")},
									[]corev1.VolumeMount{simpleVolumeMount("customv", "/customv")},
								),
							},
							Containers: []corev1.Container{
								simpleContainer2("chainer", "dummy",
									[]corev1.EnvVar{simpleEnvVar("ENV", "VAL")},
									[]corev1.VolumeMount{simpleVolumeMount("customv", "/customv")},
								),
								simpleContainer("helper", "dummy"),
							},
						},
					},
				},
			},
		},
	}

	expected := func(name string, replicas int32) *appsv1.StatefulSet {
		return &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "chj" + backends.WorkerSetSuffix + "-" + name,
				Namespace: "ch",
				Labels: map[string]string{
					backends.JobLabelKey:       "chj",
					backends.WorkersetLabelKey: "ws0",
					backends.RoleLabelKey:      backends.RoleWorkerSet,
					"custom":                   "value",
				},
				OwnerReferences: ownerReference("chj"),
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName:         "chj" + backends.WorkerSetSuffix + "-" + name,
				PodManagementPolicy: podManagementPolicy,
				Replicas:            &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"custom2":                  "value2",
						backends.JobLabelKey:       "chj",
						backends.WorkersetLabelKey: "ws0",
						backends.RoleLabelKey:      backends.RoleWorkerSet,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"custom2":                  "value2",
							backends.JobLabelKey:       "chj",
							backends.WorkersetLabelKey: "ws0",
							backends.RoleLabelKey:      backends.RoleWorkerSet,
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: "chj" + backends.ServiceAccountSuffix,
						Volumes: []corev1.Volume{
							emptyDirVolume("customv"),
							emptyDirVolume(hostfileVolumeName),
							emptyDirVolume(kubectlVolumeName),
							assetsVolume("chj"+assetsSuffix, int32(0555)),
						},
						InitContainers: []corev1.Container{
							simpleContainer2("init1", "dummy",
								[]corev1.EnvVar{simpleEnvVar("ENV", "VAL")},
								append(
									[]corev1.VolumeMount{simpleVolumeMount("customv", "/customv")},
									managedVolumeMounts...,
								),
							),
							kubectlDownloaderContainer,
						},
						Containers: []corev1.Container{
							simpleContainer2("chainer", "dummy",
								append(
									[]corev1.EnvVar{simpleEnvVar("ENV", "VAL")},
									managedEnvVars...,
								),
								append(
									[]corev1.VolumeMount{simpleVolumeMount("customv", "/customv")},
									managedVolumeMounts...,
								),
							),
							simpleContainer2("helper", "dummy", managedEnvVars, managedVolumeMounts),
						},
					},
				},
			},
		}
	}

	testCases := []testCase{
		{
			name:     "simple",
			wsName:   "ws0",
			done:     false,
			in:       inputJob,
			expected: expected("ws0", int32(2)),
		},
		{
			name:     "done",
			wsName:   "ws0",
			done:     true,
			in:       inputJob,
			expected: expected("ws0", int32(0)),
		},
	}

	for _, c := range testCases {
		actualSS := newWorkerSet(c.in, c.done, c.wsName)
		actualJSON, errActual := json.MarshalIndent(actualSS, "", "  ")
		actual := ""
		if errActual != nil {
			t.Errorf("Couldn't pretty format %v, error: %v", actualSS, errActual)
			actual = fmt.Sprintf("%+v", actualSS)
		} else {
			actual = string(actualJSON)
		}

		expectedJSON, errExpected := json.MarshalIndent(c.expected, "", "  ")
		expected := ""
		if errExpected != nil {
			t.Errorf("Couldn't pretty format %v, error: %v", c.expected, errExpected)
			expected = fmt.Sprintf("%v", c.expected)
		} else {
			expected = string(expectedJSON)
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("TestCase: %v\nWant:%+v\nGot:%+v", c.name, expected, actual)
		}
	}
}

func TestIsJobDone(t *testing.T) {
	type testCase struct {
		name     string
		in       *batchv1.Job
		expected bool
	}
	testCases := []testCase{
		{
			name:     "null should return false",
			in:       nil,
			expected: false,
		},
		{
			name: "empty Conditions should return false",
			in: &batchv1.Job{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{simpleContainer("dummy", "dummy")},
						},
					},
				},
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{},
				},
			},
			expected: false,
		},
		{
			name: "it should return true when Status:True of Type:Complete Conditions ",
			in: &batchv1.Job{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{simpleContainer("dummy", "dummy")},
						},
					},
				},
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobComplete,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "it should return true when Status:True of Type:Failed Conditions ",
			in: &batchv1.Job{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{simpleContainer("dummy", "dummy")},
						},
					},
				},
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobFailed,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, c := range testCases {
		if actual := isJobDone(c.in); c.expected != actual {
			jobStatusJSON, err := json.MarshalIndent(c.in, "", "  ")
			jobStatusStr := ""
			if err != nil {
				jobStatusStr = fmt.Sprintf("%+v", c.in)
			}
			jobStatusStr = string(jobStatusJSON)
			t.Errorf("TestCase: %v\nInput:%s\nWant:%+v\nGot:%+v", c.name, jobStatusStr, c.expected, actual)
		}
	}
}

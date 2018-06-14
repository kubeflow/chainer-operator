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
package none

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/kubeflow/chainer-operator/pkg/controllers/backends"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apisv1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func Int64(v int64) *int64 {
	return &v
}

func Bool(b bool) *bool {
	return &b
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

func TestNewMaster(t *testing.T) {
	// single
	type testCase struct {
		name     string
		in       *apisv1alpha1.ChainerJob
		expected *batchv1.Job
	}
	testCases := []testCase{
		{
			name: "single",
			in: &apisv1alpha1.ChainerJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "chj",
					Namespace: "ch",
					Labels: map[string]string{
						"custom": "value",
					},
				},
				Spec: apisv1alpha1.ChainerJobSpec{
					Master: apisv1alpha1.MasterSpec{
						ActiveDeadlineSeconds: Int64(1000),
						BackoffLimit:          apisv1alpha1.Int32(10),
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"custom2": "value2",
								},
							},
							Spec: corev1.PodSpec{
								RestartPolicy: "Never",
								Containers: []corev1.Container{
									simpleContainer("chainer", "dummy"),
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
							Containers: []corev1.Container{
								simpleContainer("chainer", "dummy"),
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
		}

		expectedJSON, errExpected := json.MarshalIndent(c.expected, "", "  ")
		expected := ""
		if errExpected != nil {
			t.Errorf("Couldn't pretty format %v, error: %v", c.expected, errExpected)
			expected = fmt.Sprintf("%v", c.expected)
		}

		actual = string(actualJSON)
		expected = string(expectedJSON)

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("TestCase: %v\nWant:%+v\nGot:%+v", c.name, expected, actual)
		}
	}
}

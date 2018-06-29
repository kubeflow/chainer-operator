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
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"

	corev1 "k8s.io/api/core/v1"
)

func TestSetDefaults_ChainerJob(t *testing.T) {
	type testCase struct {
		name     string
		in       *ChainerJob
		expected *ChainerJob
	}

	testCases := []testCase{
		{
			name: "normal case(backend, slots, container name, replicas, restartPolicy)",
			in: &ChainerJob{
				Spec: ChainerJobSpec{
					Master: MasterSpec{
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
					WorkerSets: map[string]*WorkerSetSpec{
						"ws": &WorkerSetSpec{
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
			expected: &ChainerJob{
				Spec: ChainerJobSpec{
					Backend: BackendTypeMPI,
					Master: MasterSpec{
						MPIConfig: &MPIConfig{
							Slots: Int32(DefaultSlots),
						},
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy: DefaultRestartPolicy,
								Containers: []corev1.Container{
									{
										Name:  DefaultContainerName,
										Image: "dummy",
									},
								},
							},
						},
					},
					WorkerSets: map[string]*WorkerSetSpec{
						"ws": &WorkerSetSpec{
							Replicas: Int32(1),
							MPIConfig: &MPIConfig{
								Slots: Int32(DefaultSlots),
							},
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  DefaultContainerName,
											Image: "dummy",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "only master (container name, replicas, restartPolicy)",
			in: &ChainerJob{
				Spec: ChainerJobSpec{
					Master: MasterSpec{
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
			expected: &ChainerJob{
				Spec: ChainerJobSpec{
					Master: MasterSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy: DefaultRestartPolicy,
								Containers: []corev1.Container{
									{
										Name:  DefaultContainerName,
										Image: "dummy",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "nvidia.com/gpus to be slots",
			in: &ChainerJob{
				Spec: ChainerJobSpec{
					Backend: BackendTypeMPI,
					Master: MasterSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Image: "dummy",
										Resources: corev1.ResourceRequirements{
											Limits: corev1.ResourceList{
												"nvidia.com/gpu": resource.MustParse("2"),
											},
										},
									},
								},
							},
						},
					},
					WorkerSets: map[string]*WorkerSetSpec{
						"ws": &WorkerSetSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: "dummy",
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													"nvidia.com/gpu": resource.MustParse("3"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &ChainerJob{
				Spec: ChainerJobSpec{
					Backend: BackendTypeMPI,
					Master: MasterSpec{
						MPIConfig: &MPIConfig{
							Slots: Int32(2),
						},
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy: DefaultRestartPolicy,
								Containers: []corev1.Container{
									{
										Name:  DefaultContainerName,
										Image: "dummy",
										Resources: corev1.ResourceRequirements{
											Limits: corev1.ResourceList{
												"nvidia.com/gpu": resource.MustParse("2"),
											},
										},
									},
								},
							},
						},
					},
					WorkerSets: map[string]*WorkerSetSpec{
						"ws": &WorkerSetSpec{
							MPIConfig: &MPIConfig{
								Slots: Int32(3),
							},
							Replicas: Int32(1),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  DefaultContainerName,
											Image: "dummy",
											Resources: corev1.ResourceRequirements{
												Limits: corev1.ResourceList{
													"nvidia.com/gpu": resource.MustParse("3"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "multiple containers case doensn't set default ContainerName but set default Slots",
			in: &ChainerJob{
				Spec: ChainerJobSpec{
					Backend: BackendTypeMPI,
					Master: MasterSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Image: "dummy",
									},
									{
										Image: "dummy",
									},
								},
							},
						},
					},
					WorkerSets: map[string]*WorkerSetSpec{
						"ws": &WorkerSetSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: "dummy",
										},
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
			expected: &ChainerJob{
				Spec: ChainerJobSpec{
					Backend: BackendTypeMPI,
					Master: MasterSpec{
						MPIConfig: &MPIConfig{
							Slots: Int32(DefaultSlots),
						},
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy: DefaultRestartPolicy,
								Containers: []corev1.Container{
									{
										Image: "dummy",
									},
									{
										Image: "dummy",
									},
								},
							},
						},
					},
					WorkerSets: map[string]*WorkerSetSpec{
						"ws": &WorkerSetSpec{
							MPIConfig: &MPIConfig{
								Slots: Int32(DefaultSlots),
							},
							Replicas: Int32(1),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Image: "dummy",
										},
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
		},
		{
			name: "everyghing is already filled",
			in: &ChainerJob{
				Spec: ChainerJobSpec{
					Backend: BackendTypeMPI,
					Master: MasterSpec{
						MPIConfig: &MPIConfig{
							Slots: Int32(3),
						},
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy: corev1.RestartPolicyOnFailure,
								Containers: []corev1.Container{
									{
										Name:  "dummy",
										Image: "dummy",
									},
								},
							},
						},
					},
					WorkerSets: map[string]*WorkerSetSpec{
						"ws": &WorkerSetSpec{
							Replicas: Int32(5),
							MPIConfig: &MPIConfig{
								Slots: Int32(2),
							},
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "dummy",
											Image: "dummy",
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &ChainerJob{
				Spec: ChainerJobSpec{
					Backend: BackendTypeMPI,
					Master: MasterSpec{
						MPIConfig: &MPIConfig{
							Slots: Int32(3),
						},
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy: corev1.RestartPolicyOnFailure,
								Containers: []corev1.Container{
									{
										Name:  "dummy",
										Image: "dummy",
									},
								},
							},
						},
					},
					WorkerSets: map[string]*WorkerSetSpec{
						"ws": &WorkerSetSpec{
							Replicas: Int32(5),
							MPIConfig: &MPIConfig{
								Slots: Int32(2),
							},
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "dummy",
											Image: "dummy",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, c := range testCases {
		SetDefaults_ChainerJob(c.in)
		actualJSON, errActual := json.MarshalIndent(c.in, "", "  ")
		actual := ""
		if errActual != nil {
			t.Errorf("Couldn't pretty format %v, error: %v", c.in, errActual)
			actual = fmt.Sprintf("%+v", c.in)
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

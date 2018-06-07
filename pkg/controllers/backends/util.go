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

package backends

import (
	"fmt"

	apisv1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	appslister "k8s.io/client-go/listers/apps/v1"
	batchlister "k8s.io/client-go/listers/batch/v1"
	corelister "k8s.io/client-go/listers/core/v1"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/record"

	clientset "github.com/kubeflow/chainer-operator/pkg/client/clientset/versioned"
)

func NewServiceAccount(chjob *apisv1alpha1.ChainerJob) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chjob.Name + ServiceAccountSuffix,
			Namespace: chjob.Namespace,
			Labels: map[string]string{
				JobLabelKey: chjob.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(chjob, apisv1alpha1.SchemeGroupVersionKind),
			},
		},
	}
}

func NewRole(chjob *apisv1alpha1.ChainerJob) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chjob.Name + RoleSuffix,
			Namespace: chjob.Namespace,
			Labels: map[string]string{
				JobLabelKey: chjob.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(chjob, apisv1alpha1.SchemeGroupVersionKind),
			},
		},
		Rules: []rbacv1.PolicyRule{},
	}
}

func NewRoleBindings(chjob *apisv1alpha1.ChainerJob) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chjob.Name + RolebindingSuffix,
			Namespace: chjob.Namespace,
			Labels: map[string]string{
				JobLabelKey: chjob.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(chjob, apisv1alpha1.SchemeGroupVersionKind),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      chjob.Name + ServiceAccountSuffix,
				Namespace: chjob.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     chjob.Name + RoleSuffix,
		},
	}
}

func NewMasterJob(chjob *apisv1alpha1.ChainerJob) *batchv1.Job {
	masterSpec := chjob.Spec.Master

	// manged parts to be injected.
	managedLabels := map[string]string{
		JobLabelKey:  chjob.Name,
		RoleLabelKey: RoleMaster,
	}

	// inject aboves to user defined template
	jobLabels := map[string]string{}
	for k, v := range chjob.Labels {
		jobLabels[k] = v
	}
	podTemplate := masterSpec.Template.DeepCopy()
	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string, len(managedLabels))
	}
	for k, v := range managedLabels {
		podTemplate.Labels[k] = v
		jobLabels[k] = v
	}
	podTemplate.Spec.ServiceAccountName = chjob.Name + ServiceAccountSuffix

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chjob.Name + MasterSuffix,
			Namespace: chjob.Namespace,
			Labels:    jobLabels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(chjob, apisv1alpha1.SchemeGroupVersionKind),
			},
		},
		Spec: batchv1.JobSpec{
			Completions:           apisv1alpha1.Int32(1),
			Parallelism:           apisv1alpha1.Int32(1),
			BackoffLimit:          masterSpec.BackoffLimit,
			ActiveDeadlineSeconds: masterSpec.ActiveDeadlineSeconds,
			Template:              *podTemplate,
		},
	}
}

func NewWorkerSet(chjob *apisv1alpha1.ChainerJob, done bool, name string) *appsv1.StatefulSet {
	workerSetSpec := chjob.Spec.WorkerSets[name]

	var targetReplicas *int32
	if done {
		targetReplicas = apisv1alpha1.Int32(0)
	} else {
		targetReplicas = chjob.Spec.WorkerSets[name].Replicas
	}

	// manged parts to be injected.
	managedLabels := map[string]string{
		JobLabelKey:       chjob.Name,
		RoleLabelKey:      RoleWorkerSet,
		WorkersetLabelKey: name,
	}

	// inject aboves to user defined template
	ssLabels := map[string]string{}
	for k, v := range chjob.Labels {
		ssLabels[k] = v
	}
	podTemplate := workerSetSpec.Template.DeepCopy()
	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string, len(managedLabels))
	}
	for k, v := range managedLabels {
		podTemplate.Labels[k] = v
		ssLabels[k] = v
	}
	podTemplate.Spec.ServiceAccountName = chjob.Name + ServiceAccountSuffix

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      chjob.Name + WorkerSetSuffix + "-" + name,
			Namespace: chjob.Namespace,
			Labels:    ssLabels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(chjob, apisv1alpha1.SchemeGroupVersionKind),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			PodManagementPolicy: PodManagementPolicy,
			Replicas:            targetReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podTemplate.Labels,
			},
			ServiceName: chjob.Name + WorkerSetSuffix + "-" + name,
			Template:    *podTemplate,
		},
	}
}

func CreateOrUpdateConfigMap(
	chjob *apisv1alpha1.ChainerJob,
	kubeClient kubernetes.Interface,
	cmLister corelister.ConfigMapLister,
	recorder record.EventRecorder,
	newResource func(chj *apisv1alpha1.ChainerJob) *corev1.ConfigMap,
) (*corev1.ConfigMap, error) {
	desired := newResource(chjob)
	client := kubeClient.CoreV1().ConfigMaps(desired.Namespace)
	lister := cmLister.ConfigMaps(desired.Namespace)

	rs, err := lister.Get(desired.Name)

	created := false
	if errors.IsNotFound(err) {
		rs, err = client.Create(desired)
		created = true
	}

	if err != nil {
		// If an error occurs during Get, we'll requeue the item so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		return nil, err
	}

	// If the resource is not controlled by this ChainerJob resource, we should log
	// a warning to the event recorder and return.
	if !metav1.IsControlledBy(rs, chjob) {
		msg := fmt.Sprintf(MessageResourceExists, rs.Name)
		recorder.Event(chjob, corev1.EventTypeWarning, ErrResourceExists, msg)
		return rs, fmt.Errorf(msg)
	}

	if !created {
		rs, err = client.Update(desired)
		if err != nil {
			return rs, err
		}
	}

	return rs, nil
}

func CreateServiceAccountIfNotExist(
	chjob *apisv1alpha1.ChainerJob,
	kubeClient kubernetes.Interface,
	saLister corelister.ServiceAccountLister,
	recorder record.EventRecorder,
	newResource func(chj *apisv1alpha1.ChainerJob) *corev1.ServiceAccount,
) (*corev1.ServiceAccount, error) {
	desired := newResource(chjob)
	client := kubeClient.CoreV1().ServiceAccounts(desired.Namespace)
	lister := saLister.ServiceAccounts(desired.Namespace)

	rs, err := lister.Get(desired.Name)

	if errors.IsNotFound(err) {
		rs, err = client.Create(desired)
	}

	if err != nil {
		// If an error occurs during Get, we'll requeue the item so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		return nil, err
	}

	// If the resource is not controlled by this ChainerJob resource, we should log
	// a warning to the event recorder and return.
	if !metav1.IsControlledBy(rs, chjob) {
		msg := fmt.Sprintf(MessageResourceExists, rs.Name)
		recorder.Event(chjob, corev1.EventTypeWarning, ErrResourceExists, msg)
		return rs, fmt.Errorf(msg)
	}

	return rs, nil
}

func CreateOrUpdateRole(
	chjob *apisv1alpha1.ChainerJob,
	kubeClient kubernetes.Interface,
	roleLister rbaclister.RoleLister,
	recorder record.EventRecorder,
	newResource func(chj *apisv1alpha1.ChainerJob) *rbacv1.Role,
) (*rbacv1.Role, error) {
	desired := newResource(chjob)
	client := kubeClient.RbacV1().Roles(desired.Namespace)
	lister := roleLister.Roles(desired.Namespace)

	rs, err := lister.Get(desired.Name)

	created := false
	if errors.IsNotFound(err) {
		rs, err = client.Create(desired)
		created = true
	}

	if err != nil {
		// If an error occurs during Get, we'll requeue the item so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		return nil, err
	}

	// If the resource is not controlled by this ChainerJob resource, we should log
	// a warning to the event recorder and return.
	if !metav1.IsControlledBy(rs, chjob) {
		msg := fmt.Sprintf(MessageResourceExists, rs.Name)
		recorder.Event(chjob, corev1.EventTypeWarning, ErrResourceExists, msg)
		return rs, fmt.Errorf(msg)
	}

	if !created {
		rs, err = client.Update(desired)
		if err != nil {
			return rs, err
		}
	}

	return rs, nil
}

func CreateOrUpdateRoleBinding(
	chjob *apisv1alpha1.ChainerJob,
	kubeClient kubernetes.Interface,
	roleBindingLister rbaclister.RoleBindingLister,
	recorder record.EventRecorder,
	newResource func(chj *apisv1alpha1.ChainerJob) *rbacv1.RoleBinding,
) (*rbacv1.RoleBinding, error) {
	desired := newResource(chjob)
	client := kubeClient.RbacV1().RoleBindings(desired.Namespace)
	lister := roleBindingLister.RoleBindings(desired.Namespace)

	rs, err := lister.Get(desired.Name)

	created := false
	if errors.IsNotFound(err) {
		rs, err = client.Create(desired)
		created = true
	}

	if err != nil {
		// If an error occurs during Get, we'll requeue the item so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		return nil, err
	}

	// If the resource is not controlled by this ChainerJob resource, we should log
	// a warning to the event recorder and return.
	if !metav1.IsControlledBy(rs, chjob) {
		msg := fmt.Sprintf(MessageResourceExists, rs.Name)
		recorder.Event(chjob, corev1.EventTypeWarning, ErrResourceExists, msg)
		return rs, fmt.Errorf(msg)
	}

	if !created {
		rs, err = client.Update(desired)
		if err != nil {
			return rs, err
		}
	}

	return rs, nil
}

func CreateJobIfNotExist(
	chjob *apisv1alpha1.ChainerJob,
	kubeClient kubernetes.Interface,
	jobLister batchlister.JobLister,
	recorder record.EventRecorder,
	newResource func(chj *apisv1alpha1.ChainerJob) *batchv1.Job,
) (*batchv1.Job, error) {
	desired := newResource(chjob)
	client := kubeClient.BatchV1().Jobs(desired.Namespace)
	lister := jobLister.Jobs(desired.Namespace)

	rs, err := lister.Get(desired.Name)

	if errors.IsNotFound(err) {
		rs, err = client.Create(desired)
	}

	if err != nil {
		// If an error occurs during Get, we'll requeue the item so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		return nil, err
	}

	// If the resource is not controlled by this ChainerJob resource, we should log
	// a warning to the event recorder and return.
	if !metav1.IsControlledBy(rs, chjob) {
		msg := fmt.Sprintf(MessageResourceExists, rs.Name)
		recorder.Event(chjob, corev1.EventTypeWarning, ErrResourceExists, msg)
		return rs, fmt.Errorf(msg)
	}

	return rs, nil
}

func CreateOrUpdateStatefulSet(
	chjob *apisv1alpha1.ChainerJob,
	kubeClient kubernetes.Interface,
	statefulSetLister appslister.StatefulSetLister,
	recorder record.EventRecorder,
	newResource func(chj *apisv1alpha1.ChainerJob) *appsv1.StatefulSet,
) (*appsv1.StatefulSet, error) {
	desired := newResource(chjob)
	client := kubeClient.AppsV1().StatefulSets(desired.Namespace)
	lister := statefulSetLister.StatefulSets(desired.Namespace)

	rs, err := lister.Get(desired.Name)

	created := false
	if errors.IsNotFound(err) {
		rs, err = client.Create(desired)
		created = true
	}

	if err != nil {
		// If an error occurs during Get, we'll requeue the item so we can
		// attempt processing again later. This could have been caused by a
		// temporary network failure, or any other transient reason.
		return nil, err
	}

	// If the resource is not controlled by this ChainerJob resource, we should log
	// a warning to the event recorder and return.
	if !metav1.IsControlledBy(rs, chjob) {
		msg := fmt.Sprintf(MessageResourceExists, rs.Name)
		recorder.Event(chjob, corev1.EventTypeWarning, ErrResourceExists, msg)
		return rs, fmt.Errorf(msg)
	}

	if !created {
		rs, err = client.Update(desired)
		if err != nil {
			return rs, err
		}
	}

	return rs, nil
}

func UpdateChainerJobStatus(
	chjob *apisv1alpha1.ChainerJob,
	status *batchv1.JobStatus,
	kubeflowClient clientset.Interface,
) error {
	chjobCopy := chjob.DeepCopy()
	chjobCopy.Status = *status.DeepCopy()

	_, err := kubeflowClient.KubeflowV1alpha1().ChainerJobs(chjob.Namespace).Update(chjobCopy)

	return err
}

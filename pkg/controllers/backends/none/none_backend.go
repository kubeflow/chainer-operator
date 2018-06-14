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
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes"
	batchlister "k8s.io/client-go/listers/batch/v1"
	corelister "k8s.io/client-go/listers/core/v1"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/record"

	"github.com/golang/glog"
	apisv1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
	clientset "github.com/kubeflow/chainer-operator/pkg/client/clientset/versioned"
	"github.com/kubeflow/chainer-operator/pkg/controllers/backends"
)

// Backend defines behavior of non-Disributed ChainerJob
type Backend struct {
	kubeClient           kubernetes.Interface
	kubeflowClient       clientset.Interface
	serviceAccountLister corelister.ServiceAccountLister
	roleLister           rbaclister.RoleLister
	roleBindingLister    rbaclister.RoleBindingLister
	jobLister            batchlister.JobLister
	recorder             record.EventRecorder
}

// NewBackend is a constructor for none.Backend
func NewBackend(
	kubeClient kubernetes.Interface,
	kubeflowClient clientset.Interface,
	serviceAccountLister corelister.ServiceAccountLister,
	roleLister rbaclister.RoleLister,
	roleBindingLister rbaclister.RoleBindingLister,
	jobLister batchlister.JobLister,
	recorder record.EventRecorder,
) backends.Backend {
	return &Backend{
		kubeClient:           kubeClient,
		kubeflowClient:       kubeflowClient,
		serviceAccountLister: serviceAccountLister,
		roleLister:           roleLister,
		roleBindingLister:    roleBindingLister,
		jobLister:            jobLister,
		recorder:             recorder,
	}
}

// SyncChainerJob syncs non-Distributed ChainerJobs.
func (b *Backend) SyncChainerJob(chjob *apisv1alpha1.ChainerJob) error {

	if apisv1alpha1.IsDistributed(chjob) {
		return fmt.Errorf("not syncing %s because it is a distributed job", chjob.Name)
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

	master, err := b.syncMaster(chjob)
	if master == nil || err != nil {
		return err
	}
	glog.V(4).Infof("syncing %s: job %s synced.", chjob.Name, master.Name)

	err = backends.UpdateChainerJobStatus(chjob, &master.Status, b.kubeflowClient)
	if err != nil {
		return err
	}
	glog.V(4).Infof("syncing %s: status updated.", chjob.Name)

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
		backends.NewRole,
	)
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

func (b *Backend) syncMaster(chjob *apisv1alpha1.ChainerJob) (*batchv1.Job, error) {
	return backends.CreateJobIfNotExist(
		chjob,
		b.kubeClient,
		b.jobLister,
		b.recorder,
		newMasterJob,
	)
}

func newMasterJob(chjob *apisv1alpha1.ChainerJob) *batchv1.Job {
	return backends.NewMasterJob(chjob)
}

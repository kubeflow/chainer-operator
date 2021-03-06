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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// ChainerJobLister helps list ChainerJobs.
type ChainerJobLister interface {
	// List lists all ChainerJobs in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.ChainerJob, err error)
	// ChainerJobs returns an object that can list and get ChainerJobs.
	ChainerJobs(namespace string) ChainerJobNamespaceLister
	ChainerJobListerExpansion
}

// chainerJobLister implements the ChainerJobLister interface.
type chainerJobLister struct {
	indexer cache.Indexer
}

// NewChainerJobLister returns a new ChainerJobLister.
func NewChainerJobLister(indexer cache.Indexer) ChainerJobLister {
	return &chainerJobLister{indexer: indexer}
}

// List lists all ChainerJobs in the indexer.
func (s *chainerJobLister) List(selector labels.Selector) (ret []*v1alpha1.ChainerJob, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ChainerJob))
	})
	return ret, err
}

// ChainerJobs returns an object that can list and get ChainerJobs.
func (s *chainerJobLister) ChainerJobs(namespace string) ChainerJobNamespaceLister {
	return chainerJobNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ChainerJobNamespaceLister helps list and get ChainerJobs.
type ChainerJobNamespaceLister interface {
	// List lists all ChainerJobs in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.ChainerJob, err error)
	// Get retrieves the ChainerJob from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.ChainerJob, error)
	ChainerJobNamespaceListerExpansion
}

// chainerJobNamespaceLister implements the ChainerJobNamespaceLister
// interface.
type chainerJobNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ChainerJobs in the indexer for a given namespace.
func (s chainerJobNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ChainerJob, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ChainerJob))
	})
	return ret, err
}

// Get retrieves the ChainerJob from the indexer for a given namespace and name.
func (s chainerJobNamespaceLister) Get(name string) (*v1alpha1.ChainerJob, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("chainerjob"), name)
	}
	return obj.(*v1alpha1.ChainerJob), nil
}

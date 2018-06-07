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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	chainer_v1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
	versioned "github.com/kubeflow/chainer-operator/pkg/client/clientset/versioned"
	internalinterfaces "github.com/kubeflow/chainer-operator/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kubeflow/chainer-operator/pkg/client/listers/chainer/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ChainerJobInformer provides access to a shared informer and lister for
// ChainerJobs.
type ChainerJobInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ChainerJobLister
}

type chainerJobInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewChainerJobInformer constructs a new informer for ChainerJob type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewChainerJobInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredChainerJobInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredChainerJobInformer constructs a new informer for ChainerJob type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredChainerJobInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubeflowV1alpha1().ChainerJobs(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.KubeflowV1alpha1().ChainerJobs(namespace).Watch(options)
			},
		},
		&chainer_v1alpha1.ChainerJob{},
		resyncPeriod,
		indexers,
	)
}

func (f *chainerJobInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredChainerJobInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *chainerJobInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&chainer_v1alpha1.ChainerJob{}, f.defaultInformer)
}

func (f *chainerJobInformer) Lister() v1alpha1.ChainerJobLister {
	return v1alpha1.NewChainerJobLister(f.Informer().GetIndexer())
}

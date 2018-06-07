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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
	scheme "github.com/kubeflow/chainer-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// ChainerJobsGetter has a method to return a ChainerJobInterface.
// A group's client should implement this interface.
type ChainerJobsGetter interface {
	ChainerJobs(namespace string) ChainerJobInterface
}

// ChainerJobInterface has methods to work with ChainerJob resources.
type ChainerJobInterface interface {
	Create(*v1alpha1.ChainerJob) (*v1alpha1.ChainerJob, error)
	Update(*v1alpha1.ChainerJob) (*v1alpha1.ChainerJob, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.ChainerJob, error)
	List(opts v1.ListOptions) (*v1alpha1.ChainerJobList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ChainerJob, err error)
	ChainerJobExpansion
}

// chainerJobs implements ChainerJobInterface
type chainerJobs struct {
	client rest.Interface
	ns     string
}

// newChainerJobs returns a ChainerJobs
func newChainerJobs(c *KubeflowV1alpha1Client, namespace string) *chainerJobs {
	return &chainerJobs{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the chainerJob, and returns the corresponding chainerJob object, and an error if there is any.
func (c *chainerJobs) Get(name string, options v1.GetOptions) (result *v1alpha1.ChainerJob, err error) {
	result = &v1alpha1.ChainerJob{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("chainerjobs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ChainerJobs that match those selectors.
func (c *chainerJobs) List(opts v1.ListOptions) (result *v1alpha1.ChainerJobList, err error) {
	result = &v1alpha1.ChainerJobList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("chainerjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested chainerJobs.
func (c *chainerJobs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("chainerjobs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a chainerJob and creates it.  Returns the server's representation of the chainerJob, and an error, if there is any.
func (c *chainerJobs) Create(chainerJob *v1alpha1.ChainerJob) (result *v1alpha1.ChainerJob, err error) {
	result = &v1alpha1.ChainerJob{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("chainerjobs").
		Body(chainerJob).
		Do().
		Into(result)
	return
}

// Update takes the representation of a chainerJob and updates it. Returns the server's representation of the chainerJob, and an error, if there is any.
func (c *chainerJobs) Update(chainerJob *v1alpha1.ChainerJob) (result *v1alpha1.ChainerJob, err error) {
	result = &v1alpha1.ChainerJob{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("chainerjobs").
		Name(chainerJob.Name).
		Body(chainerJob).
		Do().
		Into(result)
	return
}

// Delete takes name of the chainerJob and deletes it. Returns an error if one occurs.
func (c *chainerJobs) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("chainerjobs").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *chainerJobs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("chainerjobs").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched chainerJob.
func (c *chainerJobs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ChainerJob, err error) {
	result = &v1alpha1.ChainerJob{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("chainerjobs").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}

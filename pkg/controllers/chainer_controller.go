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

package controllers

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	batchinformers "k8s.io/client-go/informers/batch/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	rbacinformers "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	appslister "k8s.io/client-go/listers/apps/v1"
	batchlister "k8s.io/client-go/listers/batch/v1"
	corelister "k8s.io/client-go/listers/core/v1"
	rbaclister "k8s.io/client-go/listers/rbac/v1"

	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	apisv1alpha1 "github.com/kubeflow/chainer-operator/pkg/apis/chainer/v1alpha1"
	clientset "github.com/kubeflow/chainer-operator/pkg/client/clientset/versioned"
	kubeflowScheme "github.com/kubeflow/chainer-operator/pkg/client/clientset/versioned/scheme"
	informers "github.com/kubeflow/chainer-operator/pkg/client/informers/externalversions/chainer/v1alpha1"
	listers "github.com/kubeflow/chainer-operator/pkg/client/listers/chainer/v1alpha1"
	"github.com/kubeflow/chainer-operator/pkg/controllers/backends/mpi"
	"github.com/kubeflow/chainer-operator/pkg/controllers/backends/none"
)

const (
	controllerAgentName = "chainer-job-controller"

	// event types
	failedSynced  = "FailedSynced"
	successSynced = "Synced"

	// event messages
	messageBackendTypeNotSupported = "backend:%q not supported"
	messageResourceSynced          = "ChainerJob synced successfully"
)

// ChainerJobController is the controller
type ChainerJobController struct {
	// kubeClient is a standard kubernetes clientset.
	kubeClient kubernetes.Interface
	// kubeflowClient is a clientset for our own API group.
	kubeflowClient clientset.Interface

	configMapLister      corelister.ConfigMapLister
	configMapSynced      cache.InformerSynced
	serviceAccountLister corelister.ServiceAccountLister
	serviceAccountSynced cache.InformerSynced
	roleLister           rbaclister.RoleLister
	roleSynced           cache.InformerSynced
	roleBindingLister    rbaclister.RoleBindingLister
	roleBindingSynced    cache.InformerSynced
	statefulSetLister    appslister.StatefulSetLister
	statefulSetSynced    cache.InformerSynced
	jobLister            batchlister.JobLister
	jobSynced            cache.InformerSynced
	chainerJobLister     listers.ChainerJobLister
	chainerJobSynced     cache.InformerSynced

	// queue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	queue workqueue.RateLimitingInterface

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewChainerJobController returns a new ChainerJob controller.
func NewChainerJobController(
	kubeClient kubernetes.Interface,
	kubeflowClient clientset.Interface,
	configMapInformer coreinformers.ConfigMapInformer,
	serviceAccountInformer coreinformers.ServiceAccountInformer,
	roleInformer rbacinformers.RoleInformer,
	roleBindingInformer rbacinformers.RoleBindingInformer,
	statefulSetInformer appsinformers.StatefulSetInformer,
	jobInformer batchinformers.JobInformer,
	chainerJobInformer informers.ChainerJobInformer) *ChainerJobController {

	// Create event broadcaster.
	// Add chainer-job-controller types to the default Kubernetes Scheme so Events
	// can be logged for chainer-job-controller types.
	kubeflowScheme.AddToScheme(scheme.Scheme)
	glog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: controllerAgentName},
	)

	controller := &ChainerJobController{
		kubeClient:           kubeClient,
		kubeflowClient:       kubeflowClient,
		configMapLister:      configMapInformer.Lister(),
		configMapSynced:      configMapInformer.Informer().HasSynced,
		serviceAccountLister: serviceAccountInformer.Lister(),
		serviceAccountSynced: serviceAccountInformer.Informer().HasSynced,
		roleLister:           roleInformer.Lister(),
		roleSynced:           roleInformer.Informer().HasSynced,
		roleBindingLister:    roleBindingInformer.Lister(),
		roleBindingSynced:    roleBindingInformer.Informer().HasSynced,
		statefulSetLister:    statefulSetInformer.Lister(),
		statefulSetSynced:    statefulSetInformer.Informer().HasSynced,
		jobLister:            jobInformer.Lister(),
		jobSynced:            jobInformer.Informer().HasSynced,
		chainerJobLister:     chainerJobInformer.Lister(),
		chainerJobSynced:     chainerJobInformer.Informer().HasSynced,
		queue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"ChainerJobs",
		),
		recorder: recorder,
	}

	glog.Info("Setting up event handlers")
	// Set up an event handler for when ChainerJob resources change.
	chainerJobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueChainerJob,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueChainerJob(new)
		},
	})

	// Set up an event handler for when dependent resources change. This
	// handler will lookup the owner of the given resource, and if it is
	// owned by an ChainerJob resource will enqueue that ChainerJob resource for
	// processing. This way, we don't need to implement custom logic for
	// handling dependent resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newConfigMap := new.(*corev1.ConfigMap)
			oldConfigMap := old.(*corev1.ConfigMap)
			if newConfigMap.ResourceVersion == oldConfigMap.ResourceVersion {
				// Periodic re-sync will send update events for all known
				// ConfigMaps. Two different versions of the same ConfigMap
				// will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	serviceAccountInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newServiceAccount := new.(*corev1.ServiceAccount)
			oldServiceAccount := old.(*corev1.ServiceAccount)
			if newServiceAccount.ResourceVersion == oldServiceAccount.ResourceVersion {
				// Periodic re-sync will send update events for all known
				// ServiceAccounts. Two different versions of the same ServiceAccount
				// will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	roleInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newRole := new.(*rbacv1.Role)
			oldRole := old.(*rbacv1.Role)
			if newRole.ResourceVersion == oldRole.ResourceVersion {
				// Periodic re-sync will send update events for all known
				// Roles. Two different versions of the same Role
				// will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	roleBindingInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newRoleBinding := new.(*rbacv1.RoleBinding)
			oldRoleBinding := old.(*rbacv1.RoleBinding)
			if newRoleBinding.ResourceVersion == oldRoleBinding.ResourceVersion {
				// Periodic re-sync will send update events for all known
				// RoleBindings. Two different versions of the same RoleBinding
				// will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	statefulSetInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newStatefulSet := new.(*appsv1.StatefulSet)
			oldStatefulSet := old.(*appsv1.StatefulSet)
			if newStatefulSet.ResourceVersion == oldStatefulSet.ResourceVersion {
				// Periodic re-sync will send update events for all known
				// StatefulSets. Two different versions of the same StatefulSet
				// will always have different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})
	jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newJob := new.(*batchv1.Job)
			oldJob := old.(*batchv1.Job)
			if newJob.ResourceVersion == oldJob.ResourceVersion {
				// Periodic re-sync will send update events for all known Jobs.
				// Two different versions of the same Job will always have
				// different RVs.
				return
			}
			controller.handleObject(new)
		},
		DeleteFunc: controller.handleObject,
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the work queue and wait for
// workers to finish processing their current work items.
func (c *ChainerJobController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.queue.ShutDown()

	// Start the informer factories to begin populating the informer caches.
	glog.Info("Starting ChainerJob controller")

	// Wait for the caches to be synced before starting workers.
	glog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(
		stopCh,
		c.configMapSynced,
		c.serviceAccountSynced,
		c.roleSynced,
		c.roleBindingSynced,
		c.statefulSetSynced,
		c.jobSynced,
		c.chainerJobSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	glog.Info("Starting workers")
	// Launch workers to process ChainerJob resources.
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	glog.Info("Started workers")
	<-stopCh
	glog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// work queue.
func (c *ChainerJobController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the work queue and
// attempt to process it, by calling the syncHandler.
func (c *ChainerJobController) processNextWorkItem() bool {
	obj, shutdown := c.queue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.queue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the work queue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the work queue and attempted again after a back-off
		// period.
		defer c.queue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the work queue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// work queue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// work queue.
		if key, ok = obj.(string); !ok {
			// As the item in the work queue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.queue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// ChainerJob resource to be synced.
		if err := c.syncHandler(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.queue.Forget(obj)
		glog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// enqueueChainerJob takes a ChainerJob resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than ChainerJob.
func (c *ChainerJobController) enqueueChainerJob(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.queue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the ChainerJob resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that ChainerJob resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *ChainerJobController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		glog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	glog.V(4).Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a ChainerJob, we should not do anything
		// more with it.
		if ownerRef.Kind != "ChainerJob" {
			return
		}

		ChainerJob, err := c.chainerJobLister.ChainerJobs(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			glog.V(4).Infof("ignoring orphaned object '%s' of mpi job '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueueChainerJob(ChainerJob)
		return
	}
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the ChainerJob resource
// with the current status of the resource.
func (c *ChainerJobController) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name.
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	chjob, err := c.chainerJobLister.ChainerJobs(namespace).Get(name)
	// ChainerJob may no longer exists.
	if errors.IsNotFound(err) {
		runtime.HandleError(fmt.Errorf("ChainerJob '%s' in work queue no longer exists", key))
		return nil
	}
	if err != nil {
		return err
	}

	scheme.Scheme.Default(chjob)

	if apisv1alpha1.IsDistributed(chjob) {
		switch chjob.Spec.Backend {
		case apisv1alpha1.BackendTypeMPI:
			backend := mpi.NewBackend(
				c.kubeClient,
				c.kubeflowClient,
				c.serviceAccountLister,
				c.roleLister,
				c.roleBindingLister,
				c.configMapLister,
				c.jobLister,
				c.statefulSetLister,
				c.recorder,
			)
			if err := backend.SyncChainerJob(chjob); err != nil {
				return err
			}
		default:
			msg := fmt.Sprintf(messageBackendTypeNotSupported, chjob.Spec.Backend)
			c.recorder.Eventf(chjob, corev1.EventTypeWarning, failedSynced, msg)
			return fmt.Errorf(msg)
		}
	} else {
		backend := none.NewBackend(
			c.kubeClient,
			c.kubeflowClient,
			c.serviceAccountLister,
			c.roleLister,
			c.roleBindingLister,
			c.jobLister,
			c.recorder,
		)
		if err := backend.SyncChainerJob(chjob); err != nil {
			return err
		}
	}

	c.recorder.Event(chjob, corev1.EventTypeNormal, successSynced, messageResourceSynced)
	return nil
}

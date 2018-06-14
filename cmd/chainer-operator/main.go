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

package main

import (
	"flag"
	"time"

	"github.com/golang/glog"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	clientset "github.com/kubeflow/chainer-operator/pkg/client/clientset/versioned"
	informers "github.com/kubeflow/chainer-operator/pkg/client/informers/externalversions"

	"github.com/kubeflow/chainer-operator/pkg/controllers"
	"github.com/kubeflow/chainer-operator/pkg/util/signals"
)

var (
	masterURL  string
	kubeConfig string
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeConfig)
	if err != nil {
		glog.Fatalf("Error building kubeConfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	kubeflowClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building kubeflow clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	kubeflowInformerFactory := informers.NewSharedInformerFactory(kubeflowClient, time.Second*30)

	controller := controllers.NewChainerJobController(
		kubeClient,
		kubeflowClient,
		kubeInformerFactory.Core().V1().ConfigMaps(),
		kubeInformerFactory.Core().V1().ServiceAccounts(),
		kubeInformerFactory.Rbac().V1().Roles(),
		kubeInformerFactory.Rbac().V1().RoleBindings(),
		kubeInformerFactory.Apps().V1().StatefulSets(),
		kubeInformerFactory.Batch().V1().Jobs(),
		kubeflowInformerFactory.Kubeflow().V1alpha1().ChainerJobs())

	go kubeInformerFactory.Start(stopCh)
	go kubeflowInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeConfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeConfig. Only required if out-of-cluster.")
}

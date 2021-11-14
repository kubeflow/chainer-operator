## :warning: **kubeflow/chainer-operator is not maintained**

This repository has been deprecated, and will be archived soon (Nov 30th, 2021). Please consider to user [normal Jobs](https://kubernetes.io/docs/tasks/job/) for non-distributed cases and [kubeflow/mpi-controller](https://github.com/kubeflow/mpi-operator) for distributed cases.

# K8s Custom Resource and Operator For Chainer/ChainerMN jobs

__Experimental repo notice: This repository is experimental and currently only serves as a proof of concept for running distributed training with Chainer/ChainerMN on Kubernetes.__

## Overview

`ChainerJob` provides a Kubernetes custom resource that makes it easy to run distributed or non-distributed Chainer jobs on Kubernetes.

Using a Custom Resource Definition (CRD) gives users the ability to create and manage Chainer Jobs just like builtin K8s resources. For example to create a job

```bash
$ kubectl create -f examples/chainerjob.yaml
chainerjob.kubeflow.org "example-job" created
```

To list chainer jobs:

```bash
$ kubectl get chainerjobs
NAME          AGE
example-job   12s
```

## Installing the ChainerJob CRD and its operator on your k8s cluster

```bash
kubectl create -f deploy/
```

This will create:

- `ChainerJob` Custom Resource Definition (CRD)
- `chainer-operator` namespace
- RBAC related resources
  - `ServiceAccount`
  - `ClusterRole`
    - please see [`2-rbac.yaml`](deploy/2-rbac.yaml) for detailed authorized operations
  - `ClusterRoleBinding`
- `Deployment` for the chainer-operator

## Creating a ChainerJob

Once defining `ChainerJob` CRD and operator is up, you create a job by defining `ChainerJob` custom resource.

```bash
kubectl create -f examples/chainerjob-mn.yaml
```

In this case the job spec looks like this:

```yaml
apiVersion: kubeflow.org/v1alpha1
kind: ChainerJob
metadata:
  name: example-job-mn
spec:
  backend: mpi
  master:
    template:
      spec:
        containers:
        - name: chainer
          image: everpeace/chainermn:1.3.0
          command:
          - sh
          - -c
          - |
            mpiexec -n 3 -N 1 --allow-run-as-root --display-map  --mca mpi_cuda_support 0 \
            python3 /train_mnist.py -e 2 -b 1000 -u 100
  workerSets:
    ws0:
      replicas: 2
      template:
        spec:
          containers:
          - name: chainer
            image: everpeace/chainermn:1.3.0
            command:
            - sh
            - -c
            - |
              while true; do sleep 1 & wait; done
```

`ChainerJob` consists of Master/Workers.

### master

- A `ChainerJob` must have only one master
- `master` is a pod (job technically) to boot your entire distributed job.
- The pod must contain a container named `chainer`
- `master` will be restarted automatically when it failed.  You can customize retry behavior with `activeDeadlineSeconds`/`backoffLimit`.  Please see [examples/chainerjob-reference.yaml](examples/chainerjob-reference.yaml) for details.

### workerSets

- WorkerSet is a concept of a group of pods (statefulset technically) having homogeneous configurations.
- You can define multiple WorkerSets to create heterogenous workers.
- A `ChainerJob` can have 0 or more WorkerSets.
- WorkerSets can have 1 or more workers.
- Workers are automatically restarted if they exit

### backend

- `backend` define the to initiate process groups and exchange tensor data among the processes.  
- Current supported backend is `mpi`.

#### `backend: mpi`

- the operator automatically setup mpi environment among `master` and `workerSets`
- `hostfile` and required configurations will be generated automatically
- `slots=` clause in `hostfile` can be configurable.  Please see [examples/chainerjob-reference.yaml](examples/chainerjob-reference.yaml) for details.
  - Default value is 1 or the number of GPUs you requested in a container named `chainer`.

## Using GPUs

Kubernetes supports to [schedule GPUs](https://kubernetes.io/docs/tasks/manage-gpus/scheduling-gpus/) ([instructions on GKE](https://cloud.google.com/kubernetes-engine/docs/concepts/gpus)).

Once you get GPU equipped cluster, you can attach `nvidia.com/gpu` resource to your `ChainerJob` definition like this.

```yaml
apiVersion: kubeflow.org/v1alpha1
kind: ChainerJob
metadata:
  name: example-job-mn
spec:
  backend: mpi
  master:
    template:
      spec:
        containers:
        - name: chainer
          image: everpeace/chainermn:1.3.0
          resources:
            limits:
              nvidia.com/gpu: 1
      ...
```

Follow [chainer's instruction](https://docs.chainer.org/en/stable/guides/gpu.html) for using in Chainer.

## Monotring your Job

To get status of your `ChainerJob`

```yaml
$ kubectl get chainerjobs $JOB_NAME -o yaml

apiVersion: kubeflow.org/v1alpha1
kind: ChainerJob
...
status:
  completionTime: 2018-06-13T02:13:47Z
  conditions:
  - lastProbeTime: 2018-06-13T02:13:47Z
    lastTransitionTime: 2018-06-13T02:13:47Z
    status: "True"
    type: Complete
  startTime: 2018-06-13T02:04:47Z
  succeeded: 1
```

You can also list all the pods belonging `ChainerJob` by using label `chainerjob.kubeflow.org/name`.

```bash
$ kubecl get all -l chainerjob.kubeflow.org/name=example-job-mn

NAME                                 READY     STATUS    RESTARTS   AGE
pod/example-job-mn-master-jm9qw      1/1       Running   0          1m
pod/example-job-mn-workerset-ws0-0   1/1       Running   0          1m
pod/example-job-mn-workerset-ws0-1   1/1       Running   0          1m

NAME                                            DESIRED   CURRENT   AGE
statefulset.apps/example-job-mn-workerset-ws0   2         2         1m

NAME                              DESIRED   SUCCESSFUL   AGE
job.batch/example-job-mn-master   1         0            1m
```

## Access to the logs

Once you can get pod names which belongs to `ChainerJob`,  you can inspect logs in standard ways.

```bash
$ kubectl logs example-job-mn-master-jm9qw
Data for JOB [41689,1] offset 0

 ========================   JOB MAP   ========================

 Data for node: example-job-mn-master-8qvk2     Num slots: 1    Max slots: 0    Num procs: 1
        Process OMPI jobid: [41689,1] App: 0 Process rank: 0 Bound: UNBOUND

 Data for node: example-job-mn-workerset-ws0-0  Num slots: 1    Max slots: 0    Num procs: 1
        Process OMPI jobid: [41689,1] App: 0 Process rank: 1 Bound: UNBOUND

 Data for node: example-job-mn-workerset-ws0-1  Num slots: 1    Max slots: 0    Num procs: 1
        Process OMPI jobid: [41689,1] App: 0 Process rank: 2 Bound: UNBOUND

 =============================================================
Warning: using naive communicator because only naive supports CPU-only execution
Warning: using naive communicator because only naive supports CPU-only execution
Warning: using naive communicator because only naive supports CPU-only execution
==========================================
Num process (COMM_WORLD): 3
Using hierarchical communicator
Num unit: 100
Num Minibatch-size: 1000
Num epoch: 2
==========================================
epoch       main/loss   validation/main/loss  main/accuracy  validation/main/accuracy  elapsed_time
1           1.68413     0.87129               0.5325         0.807938                  10.3654
2           0.58754     0.403208              0.8483         0.884564                  16.4705
...
```


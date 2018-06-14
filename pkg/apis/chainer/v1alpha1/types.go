package v1alpha1

import (
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChainerJob describbe chainerjob info
type ChainerJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ChainerJobSpec    `json:"spec"`
	Status            batchv1.JobStatus `json:"status"`
}

// IsDistributed returns the chainerjob is distributed mode or not.
func IsDistributed(chjob *ChainerJob) bool {
	return chjob != nil && len(chjob.Spec.WorkerSets) > 0
}

// BackendType is the type of backend
type BackendType string

const (
	// BackendTypeMPI = "mpi"
	BackendTypeMPI BackendType = "mpi"
)

// ChainerJobSpec defines a spec or ChainerJob
type ChainerJobSpec struct {
	// Backend is a type of backend for how to communicate processes.
	// This is valid only when WorkerSet present.
	Backend BackendType `json:"backend,omitempty"`

	// Master is a master of the job
	Master MasterSpec `json:"master"`

	// WorkerSets is a map whose key is workerset name and value is WorkerSetSpec
	// User can define heterogeneous WorkserSets
	WorkerSets map[string]*WorkerSetSpec `json:"workerSets,omitempty"`
}

// MasterSpec defines a spec of master of mpi cluster
type MasterSpec struct {
	ActiveDeadlineSeconds *int64             `json:"activeDeadlineSeconds,omitempty"`
	BackoffLimit          *int32             `json:"backoffLimit,omitempty"`
	MPIConfig             *MPIConfig         `json:"mpiConfig,omitempty"`
	Template              v1.PodTemplateSpec `json:"template"`
}

// WorkerSetSpec defines spec of workers of mpi cluster
type WorkerSetSpec struct {
	Replicas  *int32             `json:"replicas"`
	MPIConfig *MPIConfig         `json:"mpiConfig"`
	Template  v1.PodTemplateSpec `json:"template"`
}

// MPIConfig is config object for `backend: mpi`
type MPIConfig struct {
	Slots *int32 `json:"slots,omitempty"`
}

// +resource:path=chainerjobs
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChainerJobList is a list of ChainerJob clusters.
type ChainerJobList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	// Items is a list of ChainerJobs
	Items []ChainerJob `json:"items"`
}

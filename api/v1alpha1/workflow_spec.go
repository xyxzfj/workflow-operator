package v1alpha1

import (
	"time"

	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Schedule string      `json:"schedule"`
	Steps    []*StepSpec `json:"steps"`
}

// IsDue 到（不早于）计划时间了
func (s *WorkflowSpec) IsDue() bool {
	// TODO
	return true
}

// DueAfter 多久之后到计划时间
func (s *WorkflowSpec) DueAfter() time.Duration {
	// TODO
	return 0
}

// StepSpec 工作流中的步骤。同一 Workflow 的 Step 之间串联（串行运行）。一个 Step 包含多个 Action。
// @see CronJobSpec in batch/v1/types.go
type StepSpec struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Actions     []*ActionSpec `json:"actions"`
}

// ActionSpec 步骤中的动作。同一 Step 的多个 Action 并联（并行运行）。一个 Action 包含一个 k8s Job 。
type ActionSpec struct {
	// Standard object's metadata of the jobs created from this template.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"` // TODO: use?

	Name string     `json:"name"`
	Spec v1.JobSpec `json:"spec"`
}

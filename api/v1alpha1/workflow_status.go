package v1alpha1

import (
	v1 "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	State        WorkflowState          `json:"state"`
	StepStatuses map[string]*StepStatus `json:"stepStatuses"`
}

// IsTerminated
func (s *WorkflowStatus) IsTerminated() bool {
	return s.State == WorkflowStateSucceeded || s.State == WorkflowStateFailed
}

// MarkSucceeded
func (s *WorkflowStatus) MarkSucceeded() {
	s.State = WorkflowStateSucceeded
}

// MarkFailed
func (s *WorkflowStatus) MarkFailed() {
	s.State = WorkflowStateFailed
}

func (s *WorkflowStatus) MarkRunning() {
	s.State = WorkflowStateRunning
}

// WorkflowState
type WorkflowState string

const (
	WorkflowStateInit      = "init"
	WorkflowStateRunning   = "running"
	WorkflowStateSucceeded = "succeeded"
	WorkflowStateFailed    = "failed"
)

// StepStatus
type StepStatus struct {
	Name           string                   `json:"name"`
	State          StepState                `json:"state"`
	ActionStatuses map[string]*ActionStatus `json:"actionStatuses"`
}

// MarkSucceeded
func (s *StepStatus) MarkSucceeded() {
	s.State = StepStateSucceeded
}

// MarkFailed
func (s *StepStatus) MarkFailed() {
	s.State = StepStateFailed
}

// MarkRunning
func (s *StepStatus) MarkRunning() {
	s.State = StepStateRunning
}

// StepState
type StepState string

const (
	StepStateInit      = "init"
	StepStateRunning   = "running"
	StepStateSucceeded = "succeeded"
	StepStateFailed    = "failed"
)

// ActionStatus
type ActionStatus struct {
	Name      string       `json:"name"`
	State     ActionState  `json:"state"`
	JobStatus v1.JobStatus `json:"jobStatus"`
}

// Update
func (s *ActionStatus) Update(status v1.JobStatus) {
	s.JobStatus = status

	state := ActionState(ActionStateRunning)
	for _, c := range status.Conditions {
		if c.Type == v1.JobComplete && c.Status == v12.ConditionTrue { // job succeeded
			state = ActionStateSucceeded
			break
		}
		if c.Type == v1.JobFailed && c.Status == v12.ConditionTrue { // job failed
			state = ActionStateFailed
			break
		}
	}
	s.State = state
}

// IsSucceeded
func (s *ActionStatus) IsSucceeded() bool {
	return s.State == ActionStateSucceeded
}

// IsFailed
func (s *ActionStatus) IsFailed() bool {
	return s.State == ActionStateFailed
}

// IsRunning
func (s *ActionStatus) IsRunning() bool {
	return s.State == ActionStateRunning
}

// ActionState
type ActionState string

const (
	ActionStateInit      = "init"
	ActionStateRunning   = "running"
	ActionStateSucceeded = "succeeded"
	ActionStateFailed    = "failed"
)

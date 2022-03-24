/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	workflowv1alpha1 "github.com/xyxzfj/workflow-operator/api/v1alpha1"
	"github.com/xyxzfj/workflow-operator/controllers/reconciler"
	"k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=workflow.vikivy.cn,resources=workflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workflow.vikivy.cn,resources=workflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=workflow.vikivy.cn,resources=workflows/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Workflow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	workflow, ret := r.getWorkflow(ctx, req.NamespacedName)
	if ret != nil {
		return ret.Return()
	}
	// workflow got

	// 如果当前workflow已结束（成功或失败），则返回，不再主动重试。
	if workflow.Status.IsTerminated() {
		return ctrl.Result{}, nil
	}
	// workflow not terminated

	// 如果没过指定的时间点，则返回，时间差后重试。
	// 可参考 CronJob 的实现
	// @see https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/cronjob/cronjob_controllerv2.go
	if !workflow.Spec.IsDue() {
		return ctrl.Result{RequeueAfter: workflow.Spec.DueAfter()}, nil
	}
	// it's time to exec workflow

	defer func() {
		err := r.saveWorkflowStatus(ctx, workflow)
		if err != nil {
			log.Error(err, "Update status err")
		}
	}()

	log.Info("Executing workflow", "name", req.NamespacedName)
	return r.execWorkflow(ctx, workflow)
}

func (r *WorkflowReconciler) getWorkflow(ctx context.Context,
	name types.NamespacedName) (*workflowv1alpha1.Workflow, *reconciler.Return) {

	log := ctrllog.FromContext(ctx)

	workflow := &workflowv1alpha1.Workflow{}
	err := r.Get(ctx, name, workflow)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Workflow resource not found. Ignoring since object must be deleted")
			return nil, reconciler.NewReturn(ctrl.Result{}, nil)
		}

		log.Error(err, "Failed to get Workflow")
		return nil, reconciler.NewReturn(ctrl.Result{}, err)
	}
	return workflow, nil
}

// execWorkflow 执行 workflow ，根据其 spec 。同时获取其最新状态。
func (r *WorkflowReconciler) execWorkflow(ctx context.Context,
	workflow *workflowv1alpha1.Workflow) (ctrl.Result, error) {

	log := ctrllog.FromContext(ctx)

	workflow.InitStatus()

	stepCount := len(workflow.Spec.Steps)
	for stepIndex, stepSpec := range workflow.Spec.Steps {
		log.Info(fmt.Sprintf("Checking Step %d/%d", stepIndex, stepCount))
		stepIndexedName := workflowv1alpha1.GetIndexedName(stepIndex, stepSpec.Name)
		stepStatus := workflow.Status.StepStatuses[stepIndexedName]

		// update status of all actions; create job of action if not
		actionCount := len(stepSpec.Actions)
		for actionIndex, actionSpec := range stepSpec.Actions {
			log.Info(fmt.Sprintf("Updating status of Action %d/%d of Step %d/%d",
				actionIndex, actionCount, stepIndex, stepCount))
			actionStatus := stepStatus.ActionStatuses[actionSpec.Name]

			job, ret := r.getOrCreateJob(ctx, workflow, actionSpec)
			if ret != nil {
				return ret.Return()
			}
			// job got
			actionStatus.Update(job.Status)
		} // for each action

		// update step status
		for actionIndex, actionStatus := range stepStatus.ActionStatuses {
			log.Info(fmt.Sprintf("Checking status of Action %s/%d of Step %d/%d: %s",
				actionIndex, actionCount, stepIndex, stepCount, actionStatus.State))

			switch actionStatus.State {
			case workflowv1alpha1.ActionStateInit:
				panic("Action should not be in 'init' state")
			case workflowv1alpha1.ActionStateRunning:
				stepStatus.MarkRunning()
				workflow.Status.MarkRunning()
				return ctrl.Result{}, nil
			case workflowv1alpha1.ActionStateSucceeded:
				// next action
			case workflowv1alpha1.ActionStateFailed:
				stepStatus.MarkFailed()
				workflow.Status.MarkFailed()
				return ctrl.Result{}, nil
			}
		}
		// all action succeeded
		stepStatus.MarkSucceeded()
	} // for each step
	// all steps succeeded
	workflow.Status.MarkSucceeded()

	return ctrl.Result{}, nil
}

func (r *WorkflowReconciler) getOrCreateJob(ctx context.Context,
	workflow *workflowv1alpha1.Workflow, actionSpec *workflowv1alpha1.ActionSpec) (*v1.Job, *reconciler.Return) {

	log := ctrllog.FromContext(ctx)

	job := &v1.Job{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: workflow.Namespace,
		Name:      actionSpec.Name,
	}, job)
	if err != nil {
		if errors.IsNotFound(err) { // job not found
			job, err := r.newJobForWorkflow(workflow, actionSpec)
			if err != nil {
				log.Error(err, "New Job for workflow err")
				return nil, reconciler.NewReturn(ctrl.Result{}, err)
			}
			log.Info("Creating new Job", "Namespace", job.Namespace, "Name", job.Name)
			err = r.Create(ctx, job)
			if err != nil {
				log.Error(err, "Failed to create new Job", "Namespace", job.Namespace, "Name", job.Name)
				return nil, reconciler.NewReturn(ctrl.Result{}, err)
			}
			return nil, reconciler.NewReturn(ctrl.Result{Requeue: true}, nil)
		}

		log.Error(err, "Failed to get Job")
		return nil, reconciler.NewReturn(ctrl.Result{}, err)
	}
	return job, nil
}

func (r *WorkflowReconciler) newJobForWorkflow(workflow *workflowv1alpha1.Workflow,
	actionSpec *workflowv1alpha1.ActionSpec) (*v1.Job, error) {

	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      actionSpec.Name,
			Namespace: workflow.Namespace,
		},
		Spec:   actionSpec.Spec,
		Status: v1.JobStatus{},
	}

	err := ctrl.SetControllerReference(workflow, job, r.Scheme)
	if err != nil {
		return nil, fmt.Errorf("set controller ref err: %v", err)
	}
	return job, nil
}

func (r *WorkflowReconciler) saveWorkflowStatus(ctx context.Context, workflow *workflowv1alpha1.Workflow) error {
	// TODO: update only when different
	err := r.Status().Update(ctx, workflow)
	if err != nil {
		return fmt.Errorf("update status err: %v", err)
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workflowv1alpha1.Workflow{}).
		// watch Add/Update/Delete events Jobs TODO: watch only related jobs by using predicates.
		Owns(&v1.Job{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 1}).
		Complete(r)
}

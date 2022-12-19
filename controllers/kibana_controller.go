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
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	elasticv1 "co.elastic/kibana-crd/api/v1"
)

// KibanaReconciler reconciles a Kibana object
type KibanaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type ReconcileContext struct {
	ctx    *context.Context
	req    *ctrl.Request
	logger *logr.Logger
}

var deploymentIdLabel = "elastic.co/deployment-id"
var jobRole = "elastic.co/job-role"

// TODO: add permissions for deployments here too.
//+kubebuilder:rbac:groups=elastic.co.elastic,resources=kibanas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=elastic.co.elastic,resources=kibanas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=elastic.co.elastic,resources=kibanas/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Kibana object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *KibanaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("****** reconciler loop", "req", req)

	reconcileContext := &ReconcileContext{
		ctx:    &ctx,
		logger: &logger,
		req:    &req,
	}

	var kibana elasticv1.Kibana
	if err := r.Get(ctx, req.NamespacedName, &kibana); err != nil {
		logger.Error(err, "unable to fetch Kibana")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// identifies if the deployment is currently in the process of rolling
	//
	if ShouldStartOrContinueRolling(&kibana) {
		r.synchronizeRolling(&kibana, reconcileContext)
	} else {
		// we're not in the process of a rolling, change are likely to be just config changes that
		// we should be able to reconcile in a more traditional manner
	}

	// test update status
	kibana.Status.OverallStatus = elasticv1.KibanaOverallStatusProgressing
	if err := r.Status().Update(ctx, &kibana); err != nil {
		logger.Error(err, "unable to update Kibana status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KibanaReconciler) updateStatus(ctx *context.Context, kibana *elasticv1.Kibana) error {
	return r.Status().Update(*ctx, kibana)
}

// SetupWithManager sets up the controller with the Manager.
func (r *KibanaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// mgr.GetEventRecorderFor()
	return ctrl.NewControllerManagedBy(mgr).
		For(&elasticv1.Kibana{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

func (r *KibanaReconciler) synchronizeRolling(kibana *elasticv1.Kibana, context *ReconcileContext) (ctrl.Result, error) {
	currentStage := kibana.Status.DeploymentStage
	if currentStage == elasticv1.KibanaDeployingStatusUnknown {
		currentStage = elasticv1.KibanaDeployingStatusRunningExpandJob
	}
	switch currentStage {
	case elasticv1.KibanaDeployingStatusRunningExpandJob:
		return r.reconcileExpandJobPhase(kibana, context)
	case elasticv1.KibanaDeployingStatusRollingDeployments:
		return r.reconcileRollingDeployments(kibana, context)
	case elasticv1.KibanaDeployingStatusRunningMigrateJob:
		return r.reconcileMigrateJobPhase(kibana, context)
	case elasticv1.KibanaDeployingStatusComplete:
		// TODO
	default:
		// TODO
	}

	// TODO: update status

	return ctrl.Result{}, nil
}

func (r *KibanaReconciler) reconcileExpandJobPhase(kibana *elasticv1.Kibana, context *ReconcileContext) (ctrl.Result, error) {
	logger := context.logger

	var childJobs batchv1.JobList
	if err := r.List(*context.ctx, &childJobs,
		client.InNamespace(context.req.Namespace),
		client.MatchingLabels{deploymentIdLabel: kibana.Name, jobRole: "expand-task"}); err != nil {
		context.logger.Error(err, "unable to fetch jobs")
		return ctrl.Result{}, err
	}

	// 3 scenarios here
	// a. no job => we create it
	// b. job running => we do nothing and wait
	// c. job complete => we transition to the next stage
	// note: we also need to handle edge case where job completes with an error

	if len(childJobs.Items) == 0 {
		logger.Info("*** No expand job found, creating job")
		r.startExpandPhase(kibana, context)

		// TODO: must be split at the correct layers
		kibana.Status.OverallStatus = elasticv1.KibanaOverallStatusProgressing
		kibana.Status.CurrentlyDeployingVersion = kibana.Spec.Version
		kibana.Status.DeploymentStage = elasticv1.KibanaDeployingStatusRunningExpandJob
		kibana.Status.DeploymentStageStatus = elasticv1.KibanaDeploymentStageStatusWaitingForCompletion

	} else {
		expandJob := childJobs.Items[0]
		logger.Info("*** Found expand job", "jobs", expandJob.Status)

		// TODO: naive, should check Conditions instead, or at least Failed
		if expandJob.Status.CompletionTime == nil {
			logger.Info("*** Expand job in progress, doing nothing")
		} else {
			logger.Info("*** Expand job completed, moving to next stage")
			kibana.Status.DeploymentStage = elasticv1.KibanaDeployingStatusRollingDeployments
			kibana.Status.DeploymentStageStatus = elasticv1.KibanaDeploymentStageStatusScheduled
		}

	}

	r.updateStatus(context.ctx, kibana)

	return ctrl.Result{}, nil
}

func (r *KibanaReconciler) startExpandPhase(kibana *elasticv1.Kibana, context *ReconcileContext) error {
	logger := context.logger
	expandJob := CreateExpandJob(kibana)

	if err := ctrl.SetControllerReference(kibana, expandJob, r.Scheme); err != nil {
		logger.Error(err, "error assigning controller reference")
		return err
	}

	if err := r.Create(*context.ctx, expandJob); err != nil {
		logger.Error(err, "unable to create Job for Kibana", "job", expandJob)
		return err
	}

	logger.Info("Created job", "job", expandJob)
	return nil
}

func (r *KibanaReconciler) reconcileRollingDeployments(kibana *elasticv1.Kibana, context *ReconcileContext) (ctrl.Result, error) {
	logger := context.logger

	// TODO: implement
	logger.Info("reconcileRollingDeployments - no-op for now - moving to next stage")
	kibana.Status.DeploymentStage = elasticv1.KibanaDeployingStatusRunningMigrateJob
	kibana.Status.DeploymentStageStatus = elasticv1.KibanaDeploymentStageStatusScheduled

	r.updateStatus(context.ctx, kibana)

	return ctrl.Result{}, nil
}

func (r *KibanaReconciler) reconcileMigrateJobPhase(kibana *elasticv1.Kibana, context *ReconcileContext) (ctrl.Result, error) {
	logger := context.logger

	// TODO: implement
	logger.Info("reconcileMigrateJobPhase - no-op for now - moving to next stage")
	kibana.Status.DeploymentStage = elasticv1.KibanaDeployingStatusComplete
	kibana.Status.DeploymentStageStatus = elasticv1.KibanaDeploymentStageStatusScheduled

	r.updateStatus(context.ctx, kibana)

	return ctrl.Result{}, nil
}

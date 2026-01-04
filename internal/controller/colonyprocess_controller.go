/*
Copyright 2026.

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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/colonyos/colonies/pkg/client"
	"github.com/colonyos/colonies/pkg/core"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	colonyv1 "github.com/colonyos/kolony/api/v1"
)

const (
	processFinalizer = "colony.colonyos.io/process-finalizer"
)

// ColonyProcessReconciler reconciles a ColonyProcess object
type ColonyProcessReconciler struct {
	k8sclient.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=colony.colonyos.io,resources=colonyprocesses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=colony.colonyos.io,resources=colonyprocesses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=colony.colonyos.io,resources=colonyprocesses/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *ColonyProcessReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the ColonyProcess
	var proc colonyv1.ColonyProcess
	if err := r.Get(ctx, req.NamespacedName, &proc); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Get ColonyOS client and credentials
	coloniesClient, executorPrvKey, colonyName, err := r.getColoniesClient(ctx, req.Namespace)
	if err != nil {
		log.Error(err, "Failed to create ColonyOS client")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Handle deletion
	if !proc.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&proc, processFinalizer) {
			// Cancel process if running
			if proc.Status.ProcessID != "" && !isTerminalState(proc.Status.State) {
				if err := coloniesClient.Fail(proc.Status.ProcessID, []string{"Cancelled by Kubernetes"}, executorPrvKey); err != nil {
					log.Error(err, "Failed to cancel process in ColonyOS")
				}
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(&proc, processFinalizer)
			if err := r.Update(ctx, &proc); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&proc, processFinalizer) {
		controllerutil.AddFinalizer(&proc, processFinalizer)
		if err := r.Update(ctx, &proc); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// If process is in terminal state, no need to reconcile
	if isTerminalState(proc.Status.State) {
		return ctrl.Result{}, nil
	}

	// If not submitted yet, submit to ColonyOS
	if proc.Status.ProcessID == "" {
		return r.submitProcess(ctx, &proc, coloniesClient, executorPrvKey, colonyName, log)
	}

	// Poll for status updates
	return r.pollProcessStatus(ctx, &proc, coloniesClient, executorPrvKey, log)
}

func (r *ColonyProcessReconciler) submitProcess(ctx context.Context, proc *colonyv1.ColonyProcess, coloniesClient *client.ColoniesClient, executorPrvKey, colonyName string, log logr.Logger) (ctrl.Result, error) {
	// Parse kwargs
	var kwargs map[string]interface{}
	if proc.Spec.Kwargs != nil && proc.Spec.Kwargs.Raw != nil {
		if err := json.Unmarshal(proc.Spec.Kwargs.Raw, &kwargs); err != nil {
			log.Error(err, "Failed to parse kwargs")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Build FunctionSpec
	conditions := core.Conditions{
		ColonyName:   colonyName,
		ExecutorType: proc.Spec.ExecutorType,
	}

	// Add resource conditions if specified
	if proc.Spec.Conditions != nil {
		conditions.Nodes = proc.Spec.Conditions.Nodes
		conditions.ProcessesPerNode = proc.Spec.Conditions.ProcessesPerNode
		conditions.WallTime = proc.Spec.Conditions.WallTime
		// CPU/Memory in millicores/MB format
		if proc.Spec.Conditions.CPU > 0 {
			conditions.CPU = fmt.Sprintf("%dm", proc.Spec.Conditions.CPU)
		}
		if proc.Spec.Conditions.Memory > 0 {
			conditions.Memory = fmt.Sprintf("%dMi", proc.Spec.Conditions.Memory)
		}

		if proc.Spec.Conditions.GPU != nil && proc.Spec.Conditions.GPU.Count > 0 {
			conditions.GPU = core.GPU{
				Count: proc.Spec.Conditions.GPU.Count,
				Name:  proc.Spec.Conditions.GPU.Name,
			}
			if proc.Spec.Conditions.GPU.Memory > 0 {
				conditions.GPU.Memory = fmt.Sprintf("%dMi", proc.Spec.Conditions.GPU.Memory)
			}
		}
	}

	// Default Nodes and ProcessesPerNode to 1 if not specified
	if conditions.Nodes == 0 {
		conditions.Nodes = 1
	}
	if conditions.ProcessesPerNode == 0 {
		conditions.ProcessesPerNode = 1
	}
	// Default WallTime to MaxExecTime if not specified
	if conditions.WallTime == 0 {
		conditions.WallTime = int64(proc.Spec.MaxExecTime)
	}
	// Ensure WallTime is at least 60 seconds
	if conditions.WallTime == 0 {
		conditions.WallTime = 600
	}
	// Default CPU to 1000m (1 core) if not specified
	if conditions.CPU == "" {
		conditions.CPU = "1000m"
	}
	// Default Memory to 1000Mi if not specified
	if conditions.Memory == "" {
		conditions.Memory = "1000Mi"
	}

	// Ensure MaxRetries is at least 1 for container-executor
	maxRetries := proc.Spec.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	// Convert []string to []interface{} for Args
	args := make([]interface{}, len(proc.Spec.Args))
	for i, a := range proc.Spec.Args {
		args[i] = a
	}

	funcSpec := &core.FunctionSpec{
		FuncName:    proc.Spec.FuncName,
		Args:        args,
		KwArgs:      kwargs,
		Env:         proc.Spec.Env,
		MaxWaitTime: proc.Spec.MaxWaitTime,
		MaxExecTime: proc.Spec.MaxExecTime,
		MaxRetries:  maxRetries,
		Conditions:  conditions,
	}

	// Submit process
	result, err := coloniesClient.Submit(funcSpec, executorPrvKey)
	if err != nil {
		log.Error(err, "Failed to submit process to ColonyOS")
		meta.SetStatusCondition(&proc.Status.Conditions, metav1.Condition{
			Type:               "Submitted",
			Status:             metav1.ConditionFalse,
			Reason:             "SubmitFailed",
			Message:            err.Error(),
			LastTransitionTime: metav1.Now(),
		})
		if statusErr := r.Status().Update(ctx, proc); statusErr != nil {
			log.Error(statusErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Update status with process ID
	proc.Status.ProcessID = result.ID
	proc.Status.State = colonyv1.ProcessStateWaiting
	proc.Status.SubmitTime = &metav1.Time{Time: time.Now()}

	meta.SetStatusCondition(&proc.Status.Conditions, metav1.Condition{
		Type:               "Submitted",
		Status:             metav1.ConditionTrue,
		Reason:             "Submitted",
		Message:            "Process submitted to ColonyOS",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, proc); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Submitted process to ColonyOS", "name", proc.Name, "processId", result.ID)
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *ColonyProcessReconciler) pollProcessStatus(ctx context.Context, proc *colonyv1.ColonyProcess, coloniesClient *client.ColoniesClient, executorPrvKey string, log logr.Logger) (ctrl.Result, error) {
	// Get current status from ColonyOS
	result, err := coloniesClient.GetProcess(proc.Status.ProcessID, executorPrvKey)
	if err != nil {
		log.Error(err, "Failed to get process status from ColonyOS")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Map ColonyOS state to K8s state
	var newState colonyv1.ProcessState
	switch result.State {
	case core.WAITING:
		newState = colonyv1.ProcessStateWaiting
	case core.RUNNING:
		newState = colonyv1.ProcessStateRunning
	case core.SUCCESS:
		newState = colonyv1.ProcessStateSuccess
	case core.FAILED:
		newState = colonyv1.ProcessStateFailed
	default:
		newState = colonyv1.ProcessStatePending
	}

	// Update status
	proc.Status.State = newState
	proc.Status.AssignedExecutor = result.AssignedExecutorID
	proc.Status.Retries = result.Retries

	// Convert []interface{} to []string for Output
	output := make([]string, len(result.Output))
	for i, o := range result.Output {
		if s, ok := o.(string); ok {
			output[i] = s
		} else {
			output[i] = fmt.Sprintf("%v", o)
		}
	}
	proc.Status.Output = output
	proc.Status.Errors = result.Errors

	// Parse times
	if !result.StartTime.IsZero() {
		proc.Status.StartTime = &metav1.Time{Time: result.StartTime}
	}
	if !result.EndTime.IsZero() {
		proc.Status.CompletionTime = &metav1.Time{Time: result.EndTime}
	}

	// Set appropriate conditions
	switch newState {
	case colonyv1.ProcessStateRunning:
		meta.SetStatusCondition(&proc.Status.Conditions, metav1.Condition{
			Type:               "Running",
			Status:             metav1.ConditionTrue,
			Reason:             "Running",
			Message:            fmt.Sprintf("Process running on executor %s", result.AssignedExecutorID),
			LastTransitionTime: metav1.Now(),
		})
	case colonyv1.ProcessStateSuccess:
		meta.SetStatusCondition(&proc.Status.Conditions, metav1.Condition{
			Type:               "Complete",
			Status:             metav1.ConditionTrue,
			Reason:             "Succeeded",
			Message:            "Process completed successfully",
			LastTransitionTime: metav1.Now(),
		})
	case colonyv1.ProcessStateFailed:
		errMsg := "Process failed"
		if len(result.Errors) > 0 {
			errMsg = result.Errors[0]
		}
		meta.SetStatusCondition(&proc.Status.Conditions, metav1.Condition{
			Type:               "Complete",
			Status:             metav1.ConditionTrue,
			Reason:             "Failed",
			Message:            errMsg,
			LastTransitionTime: metav1.Now(),
		})
	}

	if err := r.Status().Update(ctx, proc); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	// Determine requeue interval
	if isTerminalState(newState) {
		log.Info("Process completed", "name", proc.Name, "state", newState)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func isTerminalState(state colonyv1.ProcessState) bool {
	return state == colonyv1.ProcessStateSuccess || state == colonyv1.ProcessStateFailed
}

func (r *ColonyProcessReconciler) getColoniesClient(ctx context.Context, namespace string) (*client.ColoniesClient, string, string, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: namespace,
	}, &secret); err != nil {
		return nil, "", "", err
	}

	host := string(secret.Data["serverHost"])
	port := parsePort(string(secret.Data["serverPort"]))
	tls := string(secret.Data["tls"]) == tlsEnabledValue
	executorPrvKey := string(secret.Data["executorPrvKey"])
	colonyName := string(secret.Data["colonyName"])

	// CreateColoniesClient(host, port, insecure, skipTLSVerify)
	// insecure=true means HTTP, insecure=false means HTTPS
	insecure := !tls
	coloniesClient := client.CreateColoniesClient(host, port, insecure, false)

	return coloniesClient, executorPrvKey, colonyName, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ColonyProcessReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&colonyv1.ColonyProcess{}).
		Named("colonyprocess").
		Complete(r)
}

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
	"time"

	"github.com/colonyos/colonies/pkg/client"
	"github.com/colonyos/colonies/pkg/core"
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
	blueprintFinalizer = "colony.colonyos.io/blueprint-finalizer"
)

// BlueprintReconciler reconciles a Blueprint object
type BlueprintReconciler struct {
	k8sclient.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=colony.colonyos.io,resources=blueprints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=colony.colonyos.io,resources=blueprints/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=colony.colonyos.io,resources=blueprints/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *BlueprintReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Blueprint
	var bp colonyv1.Blueprint
	if err := r.Get(ctx, req.NamespacedName, &bp); err != nil {
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
	if !bp.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&bp, blueprintFinalizer) {
			// Delete from ColonyOS
			if bp.Status.BlueprintID != "" {
				if err := coloniesClient.RemoveBlueprint(colonyName, bp.Name, executorPrvKey); err != nil {
					log.Error(err, "Failed to delete Blueprint from ColonyOS")
				}
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(&bp, blueprintFinalizer)
			if err := r.Update(ctx, &bp); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&bp, blueprintFinalizer) {
		controllerutil.AddFinalizer(&bp, blueprintFinalizer)
		if err := r.Update(ctx, &bp); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Parse spec data
	var specData map[string]interface{}
	if bp.Spec.Data != nil && bp.Spec.Data.Raw != nil {
		if err := json.Unmarshal(bp.Spec.Data.Raw, &specData); err != nil {
			log.Error(err, "Failed to parse spec data")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// Build ColonyOS Blueprint
	// CreateBlueprint signature: (kind, name, namespace/colonyName)
	cosBp := core.CreateBlueprint(bp.Spec.Kind, bp.Name, colonyName)
	cosBp.Metadata.LocationName = bp.Spec.LocationName
	cosBp.Spec = specData

	var result *core.Blueprint
	if bp.Status.BlueprintID == "" {
		// Create new blueprint
		result, err = coloniesClient.AddBlueprint(cosBp, executorPrvKey)
		if err != nil {
			log.Error(err, "Failed to create Blueprint in ColonyOS")
			meta.SetStatusCondition(&bp.Status.Conditions, metav1.Condition{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				Reason:             "SyncFailed",
				Message:            err.Error(),
				LastTransitionTime: metav1.Now(),
			})
			if statusErr := r.Status().Update(ctx, &bp); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		// Get existing blueprint
		result, err = coloniesClient.GetBlueprint(colonyName, bp.Name, executorPrvKey)
		if err != nil {
			// Try to recreate
			result, err = coloniesClient.AddBlueprint(cosBp, executorPrvKey)
			if err != nil {
				log.Error(err, "Failed to sync Blueprint to ColonyOS")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		} else {
			// Update if spec changed
			cosBp.ID = result.ID
			cosBp.Metadata.Generation = result.Metadata.Generation + 1
			result, err = coloniesClient.UpdateBlueprint(cosBp, executorPrvKey)
			if err != nil {
				log.Error(err, "Failed to update Blueprint in ColonyOS")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}
	}

	// Update status
	bp.Status.BlueprintID = result.ID
	bp.Status.Generation = result.Metadata.Generation
	bp.Status.Synced = true
	bp.Status.LastSyncTime = &metav1.Time{Time: time.Now()}

	// Store current data from ColonyOS status
	if result.Status != nil {
		statusBytes, err := json.Marshal(result.Status)
		if err == nil {
			bp.Status.CurrentData = &runtime.RawExtension{Raw: statusBytes}
		}
	}

	meta.SetStatusCondition(&bp.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            "Blueprint synced to ColonyOS",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, &bp); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Reconciled Blueprint", "name", bp.Name, "blueprintId", result.ID, "generation", result.Metadata.Generation)
	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *BlueprintReconciler) getColoniesClient(ctx context.Context, namespace string) (*client.ColoniesClient, string, string, error) {
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
func (r *BlueprintReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&colonyv1.Blueprint{}).
		Named("blueprint").
		Complete(r)
}

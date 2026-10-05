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
	"strings"
	"time"

	"github.com/colonyos/colonies/pkg/core"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	colonyv1 "github.com/colonyos/kolony/api/v1"
)

const (
	blueprintDefinitionFinalizer = "colony.colonyos.io/blueprintdefinition-finalizer"
	credentialsSecretName        = "colonyos-credentials"
	tlsEnabledValue              = "true"
)

// BlueprintDefinitionReconciler reconciles a BlueprintDefinition object.
//
// It only uses the colonyos-credentials Secret in the definition's own namespace, never the
// default credentials Secret: definitions are colony-wide and need the colony key, so they
// belong in namespaces an administrator has provisioned with credentials.
type BlueprintDefinitionReconciler struct {
	k8sclient.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=colony.colonyos.io,resources=blueprintdefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=colony.colonyos.io,resources=blueprintdefinitions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=colony.colonyos.io,resources=blueprintdefinitions/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *BlueprintDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the BlueprintDefinition
	var def colonyv1.BlueprintDefinition
	if err := r.Get(ctx, req.NamespacedName, &def); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	resolver := &CredentialsResolver{Client: r.Client}

	// Handle deletion
	if !def.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&def, blueprintDefinitionFinalizer) {
			// Delete from ColonyOS
			err := cleanupRemote(ctx, resolver, req.Namespace, def.Status.DefinitionID, func(c *Credentials) error {
				return c.Client().RemoveBlueprintDefinition(c.ColonyName, def.Name, c.ColonyPrvKey)
			})
			if err != nil {
				log.Error(err, "Failed to resolve ColonyOS credentials for deletion")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(&def, blueprintDefinitionFinalizer)
			if err := r.Update(ctx, &def); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(&def, blueprintDefinitionFinalizer) {
		controllerutil.AddFinalizer(&def, blueprintDefinitionFinalizer)
		if err := r.Update(ctx, &def); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Get ColonyOS client and credentials
	creds, err := resolver.Resolve(ctx, req.Namespace)
	if err != nil {
		log.Error(err, "Failed to get ColonyOS credentials")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	coloniesClient, colonyPrvKey, colonyName := creds.Client(), creds.ColonyPrvKey, creds.ColonyName

	cosDef := buildBlueprintDefinition(&def, colonyName)

	var result *core.BlueprintDefinition
	if def.Status.DefinitionID == "" {
		// Create new definition
		result, err = coloniesClient.AddBlueprintDefinition(cosDef, colonyPrvKey)
		if err != nil {
			log.Error(err, "Failed to create BlueprintDefinition in ColonyOS")
			meta.SetStatusCondition(&def.Status.Conditions, metav1.Condition{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				Reason:             "SyncFailed",
				Message:            err.Error(),
				LastTransitionTime: metav1.Now(),
			})
			if statusErr := r.Status().Update(ctx, &def); statusErr != nil {
				log.Error(statusErr, "Failed to update status")
			}
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	} else {
		// Check if it exists
		result, err = coloniesClient.GetBlueprintDefinition(colonyName, def.Name, colonyPrvKey)
		if err != nil {
			// Try to recreate
			result, err = coloniesClient.AddBlueprintDefinition(cosDef, colonyPrvKey)
			if err != nil {
				log.Error(err, "Failed to sync BlueprintDefinition to ColonyOS")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
		}
	}

	// Update status
	def.Status.DefinitionID = result.ID
	def.Status.Registered = true
	def.Status.LastSyncTime = &metav1.Time{Time: time.Now()}

	meta.SetStatusCondition(&def.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		Message:            "BlueprintDefinition synced to ColonyOS",
		LastTransitionTime: metav1.Now(),
	})

	if err := r.Status().Update(ctx, &def); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Reconciled BlueprintDefinition", "name", def.Name, "definitionId", result.ID)
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// buildBlueprintDefinition converts a BlueprintDefinition CR into its ColonyOS form.
func buildBlueprintDefinition(def *colonyv1.BlueprintDefinition, colonyName string) *core.BlueprintDefinition {
	plural := def.Spec.Names.Plural
	if plural == "" {
		plural = strings.ToLower(def.Spec.Kind) + "s"
	}
	// Args: name, group, version, kind, plural, scope, executorType, functionName
	cosDef := core.CreateBlueprintDefinition(
		def.Name,
		"colony.colonyos.io",
		"v1",
		def.Spec.Kind,
		plural,
		"Namespaced",
		def.Spec.Handler.ExecutorType,
		def.Spec.Handler.FunctionName,
	)
	// The server checks colony ownership against the metadata, which the constructor leaves empty.
	cosDef.Metadata.ColonyName = colonyName
	if def.Spec.Names.Singular != "" {
		cosDef.Spec.Names.Singular = def.Spec.Names.Singular
	}
	return cosDef
}

func parsePort(s string) int {
	var port int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			port = port*10 + int(c-'0')
		}
	}
	if port == 0 {
		return 443
	}
	return port
}

// SetupWithManager sets up the controller with the Manager.
func (r *BlueprintDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&colonyv1.BlueprintDefinition{}).
		Named("blueprintdefinition").
		Complete(r)
}

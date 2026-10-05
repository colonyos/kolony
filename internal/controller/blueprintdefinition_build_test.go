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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	colonyv1 "github.com/colonyos/kolony/api/v1"
)

func TestBuildBlueprintDefinition(t *testing.T) {
	def := &colonyv1.BlueprintDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "executor-deployment"},
		Spec: colonyv1.BlueprintDefinitionSpec{
			Kind: "ExecutorDeployment",
			Names: colonyv1.BlueprintDefinitionNames{
				Singular: "executordeployment",
				Plural:   "executordeployments",
			},
			Handler: colonyv1.BlueprintDefinitionHandler{ExecutorType: "docker-reconciler", FunctionName: "reconcile"},
		},
	}

	got := buildBlueprintDefinition(def, "dev")

	if got.Metadata.ColonyName != "dev" {
		t.Errorf("Metadata.ColonyName = %q, want dev (the server rejects an empty one)", got.Metadata.ColonyName)
	}
	if got.Spec.Scope != "Namespaced" {
		t.Errorf("Spec.Scope = %q, want Namespaced", got.Spec.Scope)
	}
	if got.Spec.Names.Kind != "ExecutorDeployment" || got.Spec.Names.Plural != "executordeployments" {
		t.Errorf("unexpected names %+v", got.Spec.Names)
	}
	if got.Spec.Handler.ExecutorType != "docker-reconciler" || got.Spec.Handler.FunctionName != "reconcile" {
		t.Errorf("unexpected handler %+v", got.Spec.Handler)
	}
}

func TestBuildBlueprintDefinitionDefaultPlural(t *testing.T) {
	def := &colonyv1.BlueprintDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu"},
		Spec:       colonyv1.BlueprintDefinitionSpec{Kind: "GPUCluster"},
	}
	if got := buildBlueprintDefinition(def, "dev").Spec.Names.Plural; got != "gpuclusters" {
		t.Errorf("Plural = %q, want gpuclusters", got)
	}
}

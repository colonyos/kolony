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
	"fmt"
	"strings"

	"github.com/colonyos/colonies/pkg/client"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Credentials holds the ColonyOS connection settings read from a credentials Secret.
type Credentials struct {
	ServerHost     string
	ServerPort     int
	TLS            bool
	ColonyName     string
	ColonyPrvKey   string
	ExecutorPrvKey string

	// Source is the Secret the credentials were read from.
	Source types.NamespacedName
}

// Client creates a ColonyOS client for these credentials.
func (c *Credentials) Client() *client.ColoniesClient {
	// CreateColoniesClient(host, port, insecure, skipTLSVerify)
	// insecure=true means HTTP, insecure=false means HTTPS
	return client.CreateColoniesClient(c.ServerHost, c.ServerPort, !c.TLS, false)
}

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch

// CredentialsResolver finds the ColonyOS credentials for a namespace.
//
// A colonyos-credentials Secret in the resource's own namespace always wins. If there is none
// and Default is set, the Secret named by Default is used instead. The fallback serves tools
// that provision a fresh namespace for every request and cannot place a Secret in it first.
// BlueprintDefinitionReconciler never uses a resolver with a Default.
//
// NamespaceSelector, if set, limits the fallback to namespaces whose labels match it. Creating
// and labelling namespaces needs cluster-level rights, so this keeps the default Secret away
// from namespaces that ordinary users control.
type CredentialsResolver struct {
	Client            k8sclient.Reader
	Default           *types.NamespacedName
	NamespaceSelector labels.Selector
}

// NewCredentialsResolver builds a resolver from the operator's flags: defaultSecret is
// "<namespace>/<name>" or empty for no fallback, and namespaceSelector is a label selector
// or empty to allow the fallback in every namespace.
func NewCredentialsResolver(c k8sclient.Reader, defaultSecret, namespaceSelector string) (*CredentialsResolver, error) {
	r := &CredentialsResolver{Client: c}
	if defaultSecret != "" {
		namespace, name, ok := strings.Cut(defaultSecret, "/")
		if !ok || namespace == "" || name == "" {
			return nil, fmt.Errorf("default credentials Secret %q must be <namespace>/<name>", defaultSecret)
		}
		r.Default = &types.NamespacedName{Namespace: namespace, Name: name}
	}
	if namespaceSelector != "" {
		if r.Default == nil {
			return nil, fmt.Errorf("a namespace selector needs a default credentials Secret")
		}
		selector, err := labels.Parse(namespaceSelector)
		if err != nil {
			return nil, fmt.Errorf("invalid namespace selector %q: %w", namespaceSelector, err)
		}
		r.NamespaceSelector = selector
	}
	return r, nil
}

// Resolve returns the credentials that apply to resources in the given namespace.
func (r *CredentialsResolver) Resolve(ctx context.Context, namespace string) (*Credentials, error) {
	local := types.NamespacedName{Name: credentialsSecretName, Namespace: namespace}
	creds, err := r.read(ctx, local)
	if err == nil {
		return creds, nil
	}
	if !apierrors.IsNotFound(err) || r.Default == nil {
		return nil, err
	}

	if r.NamespaceSelector != nil {
		allowed, nsErr := r.namespaceAllowed(ctx, namespace)
		if nsErr != nil {
			return nil, nsErr
		}
		if !allowed {
			// Still NotFound, so deleting resources here is not blocked.
			return nil, fmt.Errorf("namespace %q does not match the default credentials selector %q: %w",
				namespace, r.NamespaceSelector, err)
		}
	}

	creds, defErr := r.read(ctx, *r.Default)
	if defErr != nil {
		return nil, fmt.Errorf("no %s Secret in namespace %q and default Secret %s unavailable: %w",
			credentialsSecretName, namespace, r.Default, defErr)
	}
	return creds, nil
}

// namespaceAllowed reports whether the namespace's labels match NamespaceSelector. A namespace
// that no longer exists does not match.
func (r *CredentialsResolver) namespaceAllowed(ctx context.Context, name string) (bool, error) {
	var ns corev1.Namespace
	if err := r.Client.Get(ctx, types.NamespacedName{Name: name}, &ns); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return r.NamespaceSelector.Matches(labels.Set(ns.Labels)), nil
}

func (r *CredentialsResolver) read(ctx context.Context, name types.NamespacedName) (*Credentials, error) {
	var secret corev1.Secret
	if err := r.Client.Get(ctx, name, &secret); err != nil {
		return nil, err
	}
	return &Credentials{
		ServerHost:     string(secret.Data["serverHost"]),
		ServerPort:     parsePort(string(secret.Data["serverPort"])),
		TLS:            string(secret.Data["tls"]) == tlsEnabledValue,
		ColonyName:     string(secret.Data["colonyName"]),
		ColonyPrvKey:   string(secret.Data["colonyPrvKey"]),
		ExecutorPrvKey: string(secret.Data["executorPrvKey"]),
		Source:         name,
	}, nil
}

// credentialsResolver returns resolver, or a namespace-only resolver over c when resolver is nil.
func credentialsResolver(resolver *CredentialsResolver, c k8sclient.Reader) *CredentialsResolver {
	if resolver != nil {
		return resolver
	}
	return &CredentialsResolver{Client: c}
}

// cleanupRemote runs cleanup against ColonyOS for a resource that is being deleted.
//
// Nothing is done when the resource never reached ColonyOS (empty remoteID). If no credentials
// exist for the namespace at all, the remote object is left behind and nil is returned so the
// finalizer can still be removed; otherwise a namespace whose credentials Secret is deleted
// first could never finish terminating. Any other lookup error is returned for a retry.
func cleanupRemote(ctx context.Context, resolver *CredentialsResolver, namespace, remoteID string, cleanup func(*Credentials) error) error {
	if remoteID == "" {
		return nil
	}
	log := logf.FromContext(ctx)
	creds, err := resolver.Resolve(ctx, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "No ColonyOS credentials found, leaving the remote object in ColonyOS", "remoteId", remoteID)
			return nil
		}
		return err
	}
	if err := cleanup(creds); err != nil {
		log.Error(err, "Failed to delete remote object from ColonyOS", "remoteId", remoteID)
	}
	return nil
}

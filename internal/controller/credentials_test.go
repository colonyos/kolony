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
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func credentialsSecret(namespace, name, colony string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Data: map[string][]byte{
			"serverHost":     []byte("colonies.example"),
			"serverPort":     []byte("50080"),
			"tls":            []byte("false"),
			"colonyName":     []byte(colony),
			"colonyPrvKey":   []byte("colony-key"),
			"executorPrvKey": []byte("executor-key"),
		},
	}
}

// localColony is the colony in the test Secrets that live in a resource's own namespace.
const localColony = "local"

var defaultSecret = types.NamespacedName{Namespace: "kolony", Name: "colonyos-default-credentials"}

func TestResolvePrefersNamespaceSecret(t *testing.T) {
	c := fake.NewClientBuilder().WithObjects(
		credentialsSecret("order-1", credentialsSecretName, localColony),
		credentialsSecret(defaultSecret.Namespace, defaultSecret.Name, "default"),
	).Build()
	r := &CredentialsResolver{Client: c, Default: &defaultSecret}

	creds, err := r.Resolve(context.Background(), "order-1")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.ColonyName != localColony {
		t.Errorf("ColonyName = %q, want the namespace's own Secret (%s)", creds.ColonyName, localColony)
	}
	if creds.ServerPort != 50080 || creds.TLS || creds.ExecutorPrvKey != "executor-key" {
		t.Errorf("unexpected credentials: %+v", creds)
	}
}

func TestResolveFallsBackToDefault(t *testing.T) {
	c := fake.NewClientBuilder().WithObjects(
		credentialsSecret(defaultSecret.Namespace, defaultSecret.Name, "default"),
	).Build()
	r := &CredentialsResolver{Client: c, Default: &defaultSecret}

	creds, err := r.Resolve(context.Background(), "order-2")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.ColonyName != "default" || creds.Source != defaultSecret {
		t.Errorf("got colony %q from %s, want the default Secret", creds.ColonyName, creds.Source)
	}
}

func TestResolveWithoutDefaultIsNotFound(t *testing.T) {
	r := &CredentialsResolver{Client: fake.NewClientBuilder().Build()}

	_, err := r.Resolve(context.Background(), "order-3")
	if !apierrors.IsNotFound(err) {
		t.Fatalf("err = %v, want NotFound", err)
	}
}

func TestResolveMissingDefaultIsNotFound(t *testing.T) {
	r := &CredentialsResolver{Client: fake.NewClientBuilder().Build(), Default: &defaultSecret}

	_, err := r.Resolve(context.Background(), "order-4")
	if !apierrors.IsNotFound(err) {
		t.Fatalf("err = %v, want NotFound so deletion is not blocked", err)
	}
}

func TestCleanupRemote(t *testing.T) {
	ctx := context.Background()
	withSecret := &CredentialsResolver{Client: fake.NewClientBuilder().WithObjects(
		credentialsSecret("ns", credentialsSecretName, localColony),
	).Build()}
	noSecret := &CredentialsResolver{Client: fake.NewClientBuilder().Build()}
	failing := &CredentialsResolver{Client: fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
		Get: func(context.Context, k8sclient.WithWatch, k8sclient.ObjectKey, k8sclient.Object, ...k8sclient.GetOption) error {
			return apierrors.NewServiceUnavailable("apiserver down")
		},
	}).Build()}

	tests := []struct {
		name        string
		resolver    *CredentialsResolver
		remoteID    string
		cleanupErr  error
		wantCalled  bool
		wantRequeue bool
	}{
		{name: "never synced", resolver: noSecret, remoteID: ""},
		{name: "synced", resolver: withSecret, remoteID: "id", wantCalled: true},
		{name: "remote delete fails", resolver: withSecret, remoteID: "id", cleanupErr: errors.New("boom"), wantCalled: true},
		{name: "credentials gone", resolver: noSecret, remoteID: "id"},
		{name: "transient lookup error", resolver: failing, remoteID: "id", wantRequeue: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			err := cleanupRemote(ctx, tt.resolver, "ns", tt.remoteID, func(c *Credentials) error {
				called = true
				if c.ColonyName != localColony {
					t.Errorf("cleanup got colony %q", c.ColonyName)
				}
				return tt.cleanupErr
			})
			if called != tt.wantCalled {
				t.Errorf("cleanup called = %v, want %v", called, tt.wantCalled)
			}
			if (err != nil) != tt.wantRequeue {
				t.Errorf("err = %v, want requeue %v", err, tt.wantRequeue)
			}
		})
	}
}

func namespaceWithLabels(name string, lbls map[string]string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: lbls}}
}

// provisionedLabel marks namespaces created by a provisioning tool, the ones allowed to fall back.
const provisionedLabel = "example.com/provisioned"

// selectorResolver returns a resolver that only falls back in namespaces labelled provisionedLabel.
func selectorResolver(t *testing.T, objs ...k8sclient.Object) *CredentialsResolver {
	t.Helper()
	objs = append(objs, credentialsSecret(defaultSecret.Namespace, defaultSecret.Name, "default"))
	r, err := NewCredentialsResolver(fake.NewClientBuilder().WithObjects(objs...).Build(),
		defaultSecret.String(), provisionedLabel)
	if err != nil {
		t.Fatalf("NewCredentialsResolver: %v", err)
	}
	return r
}

func TestResolveSelectorMatchingNamespaceUsesDefault(t *testing.T) {
	r := selectorResolver(t, namespaceWithLabels("order-5", map[string]string{provisionedLabel: "true"}))

	creds, err := r.Resolve(context.Background(), "order-5")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.Source != defaultSecret {
		t.Errorf("got credentials from %s, want the default Secret", creds.Source)
	}
}

func TestResolveSelectorOtherNamespaceIsNotFound(t *testing.T) {
	r := selectorResolver(t, namespaceWithLabels("team-a", map[string]string{"team": "a"}))

	_, err := r.Resolve(context.Background(), "team-a")
	if !apierrors.IsNotFound(err) {
		t.Fatalf("err = %v, want NotFound so the default Secret is not used and deletion is not blocked", err)
	}
}

func TestResolveSelectorMissingNamespaceIsNotFound(t *testing.T) {
	r := selectorResolver(t)

	_, err := r.Resolve(context.Background(), "gone")
	if !apierrors.IsNotFound(err) {
		t.Fatalf("err = %v, want NotFound", err)
	}
}

func TestResolveSelectorKeepsNamespaceSecret(t *testing.T) {
	r := selectorResolver(t,
		namespaceWithLabels("team-b", nil),
		credentialsSecret("team-b", credentialsSecretName, localColony),
	)

	creds, err := r.Resolve(context.Background(), "team-b")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if creds.ColonyName != localColony {
		t.Errorf("ColonyName = %q, want the namespace's own Secret (%s) even outside the selector", creds.ColonyName, localColony)
	}
}

func TestResolveSelectorNamespaceLookupErrorIsRetried(t *testing.T) {
	c := fake.NewClientBuilder().WithObjects(
		credentialsSecret(defaultSecret.Namespace, defaultSecret.Name, "default"),
	).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(ctx context.Context, c k8sclient.WithWatch, key k8sclient.ObjectKey, obj k8sclient.Object, opts ...k8sclient.GetOption) error {
			if _, ok := obj.(*corev1.Namespace); ok {
				return apierrors.NewServiceUnavailable("apiserver down")
			}
			return c.Get(ctx, key, obj, opts...)
		},
	}).Build()
	r, err := NewCredentialsResolver(c, defaultSecret.String(), provisionedLabel)
	if err != nil {
		t.Fatalf("NewCredentialsResolver: %v", err)
	}

	_, err = r.Resolve(context.Background(), "order-6")
	if err == nil || apierrors.IsNotFound(err) {
		t.Fatalf("err = %v, want a non-NotFound error so the reconcile is retried", err)
	}
}

func TestNewCredentialsResolver(t *testing.T) {
	tests := []struct {
		name, secret, selector string
		wantErr                bool
		wantDefault            bool
		wantSelector           bool
	}{
		{name: "no fallback"},
		{name: "default only", secret: "kolony/creds", wantDefault: true},
		{name: "default and selector", secret: "kolony/creds", selector: provisionedLabel, wantDefault: true, wantSelector: true},
		{name: "set-based selector", secret: "kolony/creds", selector: "env in (dev,test),!frozen", wantDefault: true, wantSelector: true},
		{name: "secret without namespace", secret: "creds", wantErr: true},
		{name: "secret with empty name", secret: "kolony/", wantErr: true},
		{name: "selector without default", selector: provisionedLabel, wantErr: true},
		{name: "invalid selector", secret: "kolony/creds", selector: "a in (", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewCredentialsResolver(fake.NewClientBuilder().Build(), tt.secret, tt.selector)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, want error %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if (r.Default != nil) != tt.wantDefault || (r.NamespaceSelector != nil) != tt.wantSelector {
				t.Errorf("Default = %v, NamespaceSelector = %v", r.Default, r.NamespaceSelector)
			}
		})
	}
}

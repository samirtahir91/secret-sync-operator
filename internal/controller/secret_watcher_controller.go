
/*
Copyright 2024.

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
	//"errors"
	//"os"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	syncv1 "secret-sync-operator/api/v1"
)

// SecretWatcherReconciler watches for changes to secrets in a source namespace.
type SecretWatcherReconciler struct {
    client.Client
    Namespace string
}

// RBAC
//+kubebuilder:rbac:groups=sync.samir.io,resources=secretsyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.samir.io,resources=secretsyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.samir.io,resources=secretsyncs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;update;create;delete;watch;patch

// Reconcile of SecretWatcher (source namespace secrets)
func (r *SecretWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Enter Reconcile", "req", req)
	
    // Fetch the source secret that triggered the reconcile
    sourceSecret := &corev1.Secret{}
    if err := r.Get(ctx, req.NamespacedName, sourceSecret); err != nil {
        if apierrors.IsNotFound(err) {
            // If the source secret was deleted, log it and return without error
            l.Info("Source secret not found", "namespace", req.Namespace, "name", req.Name)
            return ctrl.Result{}, nil
        }
        // For other errors, log and return the error
        l.Error(err, "Failed to get source secret", "namespace", req.Namespace, "name", req.Name)
        return ctrl.Result{}, err
    }

    // Get all secrets with the same name as the source secret across namespaces
    secrets := &corev1.SecretList{}
    listOpts := []client.ListOption{
        client.MatchingFields{"metadata.name": sourceSecret.Name},
    }
    if err := r.List(ctx, secrets, listOpts...); err != nil {
        l.Error(err, "Failed to list secrets with the same name", "name", sourceSecret.Name)
        return ctrl.Result{}, err
    }

    // Iterate over the secrets and update if owned by a SecretSync object
    for _, secret := range secrets.Items {
        if isOwnedBySecretSync(&secret) {
            l.Info("Updating secret", "namespace", secret.Namespace, "name", secret.Name)
            // Update the secret with data from the source secret
            updatedSecret := secret.DeepCopy()
            updatedSecret.Data = sourceSecret.Data

            if err := r.Update(ctx, updatedSecret); err != nil {
                l.Error(err, "Failed to update secret", "namespace", secret.Namespace, "name", secret.Name)
                return ctrl.Result{}, err
            }
            l.Info("Secret updated successfully", "namespace", secret.Namespace, "name", secret.Name)
        }
    }

    return ctrl.Result{}, nil
}

// Helper function to check if a secret is owned by a SecretSync object
func isOwnedBySecretSync(secret *corev1.Secret) bool {
    for _, ownerRef := range secret.OwnerReferences {
        if ownerRef.Kind == "SecretSync" {
            return true
        }
    }
    return false
}
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
    "errors"
    "os"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	syncv1 "secret-sync-operator/api/v1"

)

// SecretSyncReconciler reconciles a SecretSync object
type SecretSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// RBAC
//+kubebuilder:rbac:groups=sync.samir.io,resources=secretsyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=sync.samir.io,resources=secretsyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=sync.samir.io,resources=secretsyncs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;update;create;delete;watch;patch

// Check the object that triggered the reconcile and call the function for that object (secret or secretsync)
func (r *SecretSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Determine the type of object based on the Kind
    switch req.Kind.Kind {
    case "Secret":
        secret := &corev1.Secret{}
        if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
            return ctrl.Result{}, err
        }
        return r.reconcileSecret(ctx, secret)
    case "SecretSync":
        secretSync := &syncv1.SecretSync{}
        if err := r.Get(ctx, req.NamespacedName, secretSync); err != nil {
            return ctrl.Result{}, err
        }
        return r.reconcileSecretSync(ctx, secretSync)
    default:
        // Unsupported object type
        return ctrl.Result{}, nil
    }
}

// Reconciliation for Secret objects
func (r *SecretSyncReconciler) reconcileSecret(ctx context.Context, secret *corev1.Secret) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("SECRET Reconcile", "req", req)
	return ctrl.Result{}, nil
}

// Reconciliation for SecretSync objects
func (r *SecretSyncReconciler) reconcileSecretSync(ctx context.Context, secretSync *syncv1.SecretSync) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("SECRET_SYNC Reconcile", "req", req)
	return ctrl.Result{}, nil
}


// SetupWithManager sets up the controller with the Manager.
func (r *SecretSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&syncv1.SecretSync{}). // Watch changes to secretSync objects
        Owns(&corev1.Secret{}). // Watch secrets owned by SecretSync objects
        Complete(r)
}

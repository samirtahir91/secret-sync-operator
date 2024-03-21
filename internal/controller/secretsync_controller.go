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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	syncv1 "secret-sync-operator/api/v1"

	"k8s.io/apimachinery/pkg/fields" // Required for Watching
	//"k8s.io/apimachinery/pkg/types" // Required for Watching
    "sigs.k8s.io/controller-runtime/pkg/builder" // Required for Watching
    "sigs.k8s.io/controller-runtime/pkg/handler" // Required for Watching
    "sigs.k8s.io/controller-runtime/pkg/predicate" // Required for Watching
    "sigs.k8s.io/controller-runtime/pkg/reconcile" // Required for Watching
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

// Reconcile
func (r *SecretSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)
	l.Info("Enter Reconcile", "req", req)

	// Fetch the SecretSync instance
	secretSync := &syncv1.SecretSync{}
	err := r.Get(ctx, req.NamespacedName, secretSync)
	if err != nil {
		if apierrors.IsNotFound(err) {
			l.Info("SecretSync resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		l.Error(err, "Failed to get SecretSync")
		return ctrl.Result{}, err
	}

	// Read the source namespace from environment variable
	sourceNamespace := "default"
	//sourceNamespace := os.Getenv("SOURCE_NAMESPACE")
	//if sourceNamespace == "" {
	//	// Handle case where environment variable is not set
	//	return ctrl.Result{}, errors.New("SOURCE_NAMESPACE environment variable not set")
	//}

	// Call the function to delete unreferenced secrets
	if err := r.deleteUnreferencedSecrets(ctx, secretSync); err != nil {
		return ctrl.Result{}, err
	}

	// Iterate over the list of secrets specified in the CR's spec
	for _, secretName := range secretSync.Spec.Secrets {
		if err := r.syncSecret(ctx, secretSync, secretName, sourceNamespace); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Defer function to update status
	defer func() {
		// Set the sync status based on the presence of error
		syncStatus := true
		if err != nil {
			syncStatus = false
		}
		secretSync.Status.Synced = syncStatus
		// Update the status of the SecretSync resource with optimistic locking
		if err := r.Status().Update(ctx, secretSync); err != nil {
			// Handle conflict error due to outdated version
			if apierrors.IsConflict(err) {
				l.Info("Conflict: SecretSync resource has been modified, retrying...")
			}
			l.Error(err, "Unable to update secretSync's status", "status", syncStatus)
		} else {
			l.Info("secretSync's status updated", "status", syncStatus)
		}
	}()

	return ctrl.Result{}, nil
}

func (r *SecretSyncReconciler) syncSecret(ctx context.Context, secretSync *syncv1.SecretSync, secretName, sourceNamespace string) error {
	l := log.FromContext(ctx)
	l.Info("Processing", "Namespace", sourceNamespace, "Secret", secretName)

	// Get the source secret
	sourceSecret := &corev1.Secret{}
	sourceSecretKey := client.ObjectKey{
		Namespace: sourceNamespace,
		Name:      secretName,
	}
	if err := r.Get(ctx, sourceSecretKey, sourceSecret); err != nil {
		if apierrors.IsNotFound(err) {
			l.Info("Source secret not found", "Namespace", sourceNamespace, "Secret", secretName)
			return nil
		}
		l.Error(err, "Failed to get source secret", "Namespace", sourceNamespace, "Secret", secretName)
		return err
	}

	// Create or update the destination secret
	destinationSecret := &corev1.Secret{}
	destinationSecretKey := client.ObjectKey{
		Namespace: secretSync.Namespace,
		Name:      sourceSecret.Name,
	}
	if err := r.Get(ctx, destinationSecretKey, destinationSecret); err != nil {
		if apierrors.IsNotFound(err) {
			// Create the destination secret
			return r.createDestinationSecret(ctx, secretSync, sourceSecret)
		}
		l.Error(err, "Failed to get destination secret", "Namespace", secretSync.Namespace, "Secret", secretName)
		return err
	}

	// Check if the data of the source and destination secrets are different
	if !reflect.DeepEqual(sourceSecret.Data, destinationSecret.Data) {
		// Update the destination secret
		return r.updateDestinationSecret(ctx, secretSync, destinationSecret, sourceSecret)
	}
	// Destination secret is already up to date
	l.Info("Destination secret is already up to date", "Namespace", secretSync.Namespace, "Secret", secretName)
	return nil
}

func (r *SecretSyncReconciler) createDestinationSecret(ctx context.Context, secretSync *syncv1.SecretSync, sourceSecret *corev1.Secret) error {
	l := log.FromContext(ctx)
	l.Info("Creating Secret in destination namespace", "Namespace", secretSync.Namespace, "Secret", sourceSecret.Name)

	destinationSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sourceSecret.Name,
			Namespace: secretSync.Namespace,
		},
		Data: sourceSecret.Data, // Copy data from source to destination
	}
	// Set owner reference to SecretSync object
	if err := controllerutil.SetControllerReference(secretSync, destinationSecret, r.Scheme); err != nil {
		l.Error(err, "Failed to set owner reference for destination secret")
		return err
	}
	if err := r.Create(ctx, destinationSecret); err != nil {
		l.Error(err, "Failed to create Secret in the destination namespace", "Namespace", secretSync.Namespace, "Secret", sourceSecret.Name)
		return err
	}
	return nil
}

func (r *SecretSyncReconciler) updateDestinationSecret(ctx context.Context, secretSync *syncv1.SecretSync, destinationSecret, sourceSecret *corev1.Secret) error {
	l := log.FromContext(ctx)
	l.Info("Updating Secret in destination namespace", "Namespace", secretSync.Namespace, "Secret", sourceSecret.Name)

	destinationSecret.Data = sourceSecret.Data // Update data from source to destination
	// Set owner reference to SecretSync object
	if err := controllerutil.SetControllerReference(secretSync, destinationSecret, r.Scheme); err != nil {
		l.Error(err, "Failed to set owner reference for destination secret")
		return err
	}
	if err := r.Update(ctx, destinationSecret); err != nil {
		l.Error(err, "Failed to update Secret in the destination namespace", "Namespace", secretSync.Namespace, "Secret", sourceSecret.Name)
		return err
	}
	return nil
}

// Delete unreferenced secrets owned by the SecretSync object
func (r *SecretSyncReconciler) deleteUnreferencedSecrets(ctx context.Context, secretSync *syncv1.SecretSync) error {
	l := log.FromContext(ctx)

	// Fetch secrets from the source namespace (same as SecretSync namespace)
	sourceSecrets := &corev1.SecretList{}
	err := r.List(ctx, sourceSecrets, client.InNamespace(secretSync.Namespace))
	if err != nil {
		return err
	}

	// Track secrets referenced by the SecretSync object
	referencedSecrets := make(map[string]struct{})
	for _, secretName := range secretSync.Spec.Secrets {
		referencedSecrets[secretName] = struct{}{}
	}

	// Delete unreferenced secrets
	for _, secret := range sourceSecrets.Items {
		if _, exists := referencedSecrets[secret.Name]; !exists && metav1.IsControlledBy(&secret, secretSync) {
			// Secret exists in cluster, is not in the SecretSync object's list of secrets,
			// and is owned by the SecretSync object, delete it
			l.Info("Deleting unreferenced secret", "Namespace", secret.Namespace, "Name", secret.Name)
			if err := r.Delete(ctx, &secret); err != nil {
				l.Error(err, "Failed to delete unreferenced secret", "Namespace", secret.Namespace, "Name", secret.Name)
				return err
			}
			l.Info("Deleted unreferenced secret", "Namespace", secret.Namespace, "Name", secret.Name)
		}
	}

	return nil
}

const (
    secretField = ".spec.secrets"
)

// Get SecretSyncs that reference the Secret from a source namespace and trigger reconcile for each affected
func (r *SecretSyncReconciler) findObjectsForSecret(ctx context.Context, o client.Object) []reconcile.Request {
	l := log.FromContext(ctx)

    // Convert the client.Object to a Secret object
    secret, ok := o.(*corev1.Secret)
    if !ok {
        // Not a Secret object
        return nil
    }

    // Prepare a list of SecretSync objects referencing the updated secret
    secretSyncList := &syncv1.SecretSyncList{}
    listOpts := &client.ListOptions{
        FieldSelector: fields.OneTermEqualSelector(secretField, secret.GetName()),
    }
    if err := r.List(context.Background(), secretSyncList, listOpts); err != nil {
        l.Error(err, "Failed to list SecretSync objects referencing the secret", "Secret", secret.GetName())
        return nil
    }

    // Extract reconcile requests from the found SecretSync objects
    var requests []reconcile.Request
    for _, ss := range secretSyncList.Items {
        requests = append(requests, reconcile.Request{
            NamespacedName: client.ObjectKey{
                Name:      ss.GetName(),
                Namespace: ss.GetNamespace(),
            },
        })
    }

    l.Info("Retrieved SecretSync objects referencing the secret", "Secret", secret.GetName(), "ReconcileRequests", requests)
    return requests
}



// SetupWithManager sets up the controller with the Manager.
func (r *SecretSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
    if err := mgr.GetFieldIndexer().IndexField(context.Background(), &syncv1.SecretSync{}, secretField, func(rawObj client.Object) []string {
        secretSync := rawObj.(*syncv1.SecretSync)
        return secretSync.Spec.Secrets
    }); err != nil {
        return err
    }

    return ctrl.NewControllerManagedBy(mgr).
        For(&syncv1.SecretSync{}).
        Owns(&corev1.Secret{}).
        Watches(
            &corev1.Secret{},
            handler.EnqueueRequestsFromMapFunc(r.findObjectsForSecret),
            builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
            builder.WithPredicates(
                predicate.Funcs{
                    CreateFunc: func(e event.CreateEvent) bool {
                        return e.Meta.GetNamespace() == "default"
                    },
                    UpdateFunc: func(e event.UpdateEvent) bool {
                        return e.MetaNew.GetNamespace() == "default"
                    },
                    DeleteFunc: func(e event.DeleteEvent) bool {
                        return e.Meta.GetNamespace() == "default"
                    },
                    GenericFunc: func(e event.GenericEvent) bool {
                        return e.Meta.GetNamespace() == "default"
                    },
                },
            ),
        ).
        Complete(r)
}
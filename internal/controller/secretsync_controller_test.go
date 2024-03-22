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
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	syncv1 "secret-sync-operator/api/v1"
)

var (
	secret1Array        = [2]string{"secret-1", "secret-2"}
	deletedSecret1Array = [1]string{"secret-1"}
)

var _ = Describe("SecretSync controller", func() {

	const (
		secretSyncName1      = "secret-sync-1"
		secret1              = "secret-1"
		secret2              = "secret-2"
		sourceNamespace      = "default"
		destinationNamespace = "foo"
	)

	Context("When setting up the test environment", func() {
		It("Should create SecretSync custom resources", func() {
			By("Creating a namespace for the destinationNamespace")
			ctx := context.Background()
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: destinationNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			By("Creating the secret-1 in the sourceNamespace")
			secret1Obj := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret1,
					Namespace: sourceNamespace,
				},
				Data: map[string][]byte{"user": []byte("Zm9vcw==")},
			}
			Expect(k8sClient.Create(ctx, &secret1Obj)).Should(Succeed())

			By("Creating the secret-2 in the sourceNamespace")
			secret2Obj := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret2,
					Namespace: sourceNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, &secret2Obj)).Should(Succeed())

			By("Creating a first SecretSync custom resource in the destinationNamespace")
			secretSync1 := syncv1.SecretSync{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretSyncName1,
					Namespace: destinationNamespace,
				},
				Spec: syncv1.SecretSyncSpec{
					Secrets: secret1Array[:],
				},
			}
			Expect(k8sClient.Create(ctx, &secretSync1)).Should(Succeed())
		})
	})

	Context("When reconciling a SecretSync", func() {
		It("Should sync the secrets and set the SecretSync status to true", func() {
			By("Checking if the SecretSync status has changed to the right status")
			ctx := context.Background()
			// Retrieve the SecretSync object to check its status
			key := types.NamespacedName{Name: secretSyncName1, Namespace: destinationNamespace}
			retrievedSecretSync := &syncv1.SecretSync{}
			timeout := 60 * time.Second
			interval := 5 * time.Second
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, key, retrievedSecretSync); err != nil {
					return false
				}
				// Check if the Secrets field matches the expected value
				return reflect.DeepEqual(retrievedSecretSync.Status.Synced, true)
			}, timeout, interval).Should(BeTrue(), "SecretSync status didn't change to the right status")
		})
	})

	Context("When removing a secret from a SecretSync objects spec.secrets", func() {
		It("Should delete the secret in the destination namespace", func() {
			By("Getting secret2 from the destination namespace")
			ctx := context.Background()
			// Attempt to retrieve the secret secret2 from the destination namespace
			secretKey := types.NamespacedName{Name: secret2, Namespace: destinationNamespace}
			retrievedSecret := &corev1.Secret{}
			err := k8sClient.Get(ctx, secretKey, retrievedSecret)
			if err != nil {
				// An unexpected error occurred
				Fail(fmt.Sprintf("Failed to retrieve the secret %s from the destination namespace: %v", secret2, err))
				return
			}
			By("Removing a secret from a SecretSync objects spec.secrets")
			// Retrieve the SecretSync object to check its status
			key := types.NamespacedName{Name: secretSyncName1, Namespace: destinationNamespace}
			retrievedSecretSync := &syncv1.SecretSync{}
			// Retrieve the SecretSync object from the Kubernetes API server
			Expect(k8sClient.Get(ctx, key, retrievedSecretSync)).Should(Succeed())
			// Modify the SecretSync objects spec.secrets
			retrievedSecretSync.Spec.Secrets = deletedSecret1Array[:]
			// Update the SecretSync object with the modified fields
			Expect(k8sClient.Update(ctx, retrievedSecretSync)).Should(Succeed())

			By("Checking secret2 has been removed from the destination namespace")
			// Ensure the secret is deleted
			timeout := 60 * time.Second
			interval := 5 * time.Second
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretKey, retrievedSecret)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue(), "Failed to delete the secret within timeout")
		})
	})

	Context("When deleting a secret owned by a SecretSync object in a destination namespace", func() {
		It("Should trigger the reconciler to re-create the secret in the destination namespace", func() {
			By("Removing a secret from the destination namespace")
			ctx := context.Background()
			// Get the secret from the destination namespace
			secretKey := types.NamespacedName{Name: secret1, Namespace: destinationNamespace}
			retrievedSecret := &corev1.Secret{}
			err := k8sClient.Get(ctx, secretKey, retrievedSecret)
			if err != nil {
				// An unexpected error occurred
				Fail(fmt.Sprintf("Failed to retrieve the secret %s from the destination namespace: %v", secret1, err))
				return
			}
			// Record the resource version of the secret before reconciliation
			initialResourceVersion := retrievedSecret.GetResourceVersion()
			// Delete the secret
			err = k8sClient.Delete(ctx, retrievedSecret)
			if err != nil {
				// Failed to delete the secret
				Fail(fmt.Sprintf("Failed to delete the secret %s from the destination namespace: %v", secret1, err))
				return
			}

			By("Waiting for the secret to be re-created")
			timeout := 120 * time.Second
			interval := 10 * time.Second
			Eventually(func() bool {
				// Retrieve the secret again after reconciliation
				err := k8sClient.Get(ctx, secretKey, retrievedSecret)
				if err != nil {
					return false
				}
				// Compare the resource version of the secret after reconciliation
				return retrievedSecret.GetResourceVersion() != initialResourceVersion
			}, timeout, interval).Should(BeTrue(), "Failed to recreate the secret within timeout")
			// At this point, the secret has been successfully recreated
		})
	})

	Context("When modifying a secret owned by a SecretSync object in a destination namespace", func() {
		It("Should trigger the reconciler to restore the secret to its original data", func() {
			By("Modifying the data of secret1 in the destination namespace")
			ctx := context.Background()
			// Get the secret from the destination namespace
			secretKey := types.NamespacedName{Name: secret1, Namespace: destinationNamespace}
			retrievedSecret := &corev1.Secret{}
			err := k8sClient.Get(ctx, secretKey, retrievedSecret)
			if err != nil {
				// An unexpected error occurred
				Fail(fmt.Sprintf("Failed to retrieve the secret %s from the destination namespace: %v", secret1, err))
				return
			}

			// Make a copy of the original data
			originalData := retrievedSecret.Data

			// Modify the data of the secret
			modifiedData := map[string][]byte{"user": []byte("YmFycw==")}
			retrievedSecret.Data = modifiedData
			err = k8sClient.Update(ctx, retrievedSecret)
			if err != nil {
				// Failed to update the secret
				Fail(fmt.Sprintf("Failed to modify the data of secret %s in the destination namespace: %v", secret1, err))
				return
			}

			By("Waiting for the reconciler to restore the secret to its original data")
			timeout := 120 * time.Second
			interval := 5 * time.Second
			Eventually(func() bool {
				// Retrieve the secret again after reconciliation
				err := k8sClient.Get(ctx, secretKey, retrievedSecret)
				if err != nil {
					return false
				}
				// Check if the secret data has been restored to its original state
				return reflect.DeepEqual(originalData, retrievedSecret.Data)
			}, timeout, interval).Should(BeTrue(), "Failed to restore the secret to its original data within timeout")
			// At this point, the secret has been successfully restored to its original data
		})
	})

	Context("When modifying a secret in the source namespace that is referenced by one or more SecretSyncs", func() {
		It("Should trigger the reconciler to update the secret in the SecretSyncs namespaces", func() {
			By("Modifying the data of secret1 in the source namespace")
			ctx := context.Background()
			// Get the secret from the destination namespace
			secretKey := types.NamespacedName{Name: secret1, Namespace: sourceNamespace}
			retrievedSecret := &corev1.Secret{}
			err := k8sClient.Get(ctx, secretKey, retrievedSecret)
			if err != nil {
				// An unexpected error occurred
				Fail(fmt.Sprintf("Failed to retrieve the secret %s from the sourceNamespace namespace: %v", secret1, err))
				return
			}
			// Modify the data of the secret
			modifiedData := map[string][]byte{"user": []byte("YmFycw==")}
			retrievedSecret.Data = modifiedData
			err = k8sClient.Update(ctx, retrievedSecret)
			if err != nil {
				// Failed to update the secret
				Fail(fmt.Sprintf("Failed to modify the data of secret %s in the sourceNamespace namespace: %v", secret1, err))
				return
			}
			By("Waiting for the reconciler to update the secret to its updated data")
			timeout := 120 * time.Second
			interval := 5 * time.Second
			destSecretKey := types.NamespacedName{Name: secret1, Namespace: destinationNamespace}
			destretrievedSecret := &corev1.Secret{}
			Eventually(func() bool {
				// Retrieve the secret again after reconciliation
				err := k8sClient.Get(ctx, destSecretKey, destretrievedSecret)
				if err != nil {
					return false
				}
				// Check if the secret data has been updated
				return reflect.DeepEqual(retrievedSecret.Data, destretrievedSecret.Data)
			}, timeout, interval).Should(BeTrue(), "Failed to restore the secret to its original data within timeout")
			// At this point, the secret has been successfully synced
		})
	})
})

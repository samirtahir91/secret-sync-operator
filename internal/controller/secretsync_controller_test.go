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
    "os"
    "reflect"
    "time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
    "k8s.io/apimachinery/pkg/api/errors"
	syncv1 "secret-sync-operator/api/v1"
)

var (
	secret1Array = [2]string{"secret-1", "secret-2"}
    deletedSecret1Array = [1]string{"secret-1"}
)

var _ = Describe("SecretSync controller", func() {

	const (
		secretSyncName1 = "secret-sync-1"
		secret1 = "secret-1"
        secret2 = "secret-2"
		sourceNamespace = "default"
        destinationNamespace = "foo"
	)

	Context("When setting up the test environment", func() {
		It("Should create SecretSync custom resources", func() {
			By("Setting the SOURCE_NAMESPACE environment variable")
			os.Setenv("SOURCE_NAMESPACE", sourceNamespace)

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

    Context("When deleting a secret owned by a SecretSync object", func() {
        It("Should delete the secret in the destination namespace", func() {
            By("Removing a secret from a SecretSync object")
            ctx := context.Background()
            // Retrieve the SecretSync object to check its status
            key := types.NamespacedName{Name: secretSyncName1, Namespace: destinationNamespace}
            retrievedSecretSync := &syncv1.SecretSync{}
            // Retrieve the SecretSync object from the Kubernetes API server
            Expect(k8sClient.Get(ctx, key, retrievedSecretSync)).Should(Succeed())
                // Modify the SecretSync object
            retrievedSecretSync.Spec.Secrets = deletedSecret1Array[:]
                // Update the SecretSync object with the modified fields
            Expect(k8sClient.Update(ctx, retrievedSecretSync)).Should(Succeed())

            By("Checking secret2 has been removed from the destination namespace")
            // Attempt to retrieve the secret secret2 from the destination namespace
            retrievedSecret := &corev1.Secret{}
            secretKey := types.NamespacedName{Name: secret2, Namespace: destinationNamespace}
            err := k8sClient.Get(ctx, secretKey, retrievedSecret)
            // Check if the error is of type NotFound indicating that the secret has been removed
            if apierrors.IsNotFound(err) {
                // The secret has been successfully removed
                return
            } else if err != nil {
                // An unexpected error occurred
                Fail(fmt.Sprintf("Failed to retrieve the secret %s from the destination namespace: %v", secret2, err))
            }
            // The secret 'secret-1' still exists in the destination namespace
            Fail(fmt.Sprintf("The secret %s still exists in the destination namespace", secret2))
        })
    })

})
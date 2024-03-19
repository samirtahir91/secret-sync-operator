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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	tutorialv1 "my.domain/tutorial/api/v1"
)

var _ = Describe("SecretSync controller", func() {

	const (
		secretSyncName1 = "secret-sync-1"
		secret1 = "secret-1"


		secretSyncName2   = "secret-sync-2"
		secret2 = "secret-2"

		sourceNamespace = "default"
        destinationNamespace = "foo"
	)

	Context("When setting up the test environment", func() {
		It("Should create SecretSync custom resources", func() {
			By("Creating the secret")
			ctx := context.Background()
            secret := corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secret1,
					Namespace: sourceNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, &secret)).Should(Succeed())

			By("Creating a first SecretSync custom resource")
			secretSync1 := tutorialv1.SecretSync{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretSyncName1,
					Namespace: destinationNamespace,
				},
				Spec: tutorialv1.SecretSyncSpec{
					Name: secret1,
				},
			}
			Expect(k8sClient.Create(ctx, &secretSync1)).Should(Succeed())
        
			By("Creating another SecretSync custom resource")
			secretSync2 := tutorialv1.SecretSync{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretSyncName2,
					Namespace: destinationNamespace,
				},
				Spec: tutorialv1.SecretSyncSpec{
					Name: secret2,
				},
			}
			Expect(k8sClient.Create(ctx, &secretSync2)).Should(Succeed())
		})
	})
})
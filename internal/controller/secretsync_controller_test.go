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
	syncv1 "secret-sync-operator/api/v1"
)

var (
	secret1Array = [1]string{"secret-1"}
	secret2Array = [1]string{"secret-2"}
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
			By("Creating a namespace for the destinationNamespace")
			ctx := context.Background()
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: destinationNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			By("Creating a first SecretSync custom resource")
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
})
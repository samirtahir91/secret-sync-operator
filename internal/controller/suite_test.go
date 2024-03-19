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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	// Import your controller package
	syncv1 "secret-sync-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("SecretSync controller", func() {
	var (
		// Define objects needed for the tests
		ctx    context.Context
		cli    client.Client
		scheme *runtime.Scheme
		log    = log.NullLogger{}
	)

	BeforeEach(func() {
		// Setup TestEnv
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		// Create a fake client
		cli = fake.NewClientBuilder().WithScheme(scheme).Build()
	})

	Context("Reconcile", func() {
		It("should create destination secret", func() {
			// Define a SecretSync object
			secretSync := &syncv1.SecretSync{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret-sync",
					Namespace: "default",
				},
				Spec: syncv1.SecretSyncSpec{
					Secrets: []string{"test-secret"},
				},
			}
			Expect(cli.Create(ctx, secretSync)).To(Succeed())

			// Define a source secret
			sourceSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{"key": []byte("value")},
			}
			Expect(cli.Create(ctx, sourceSecret)).To(Succeed())

			// Create a controller and reconcile
			controller := &controllers.SecretSyncReconciler{
				Client: cli,
				Scheme: scheme,
			}
			Expect(controller.Reconcile(ctx, reconcile.Request{
				NamespacedName: client.ObjectKeyFromObject(secretSync),
			})).To(Succeed())

			// Verify destination secret is created
			destinationSecret := &corev1.Secret{}
			Expect(cli.Get(ctx, client.ObjectKey{Name: "test-secret", Namespace: "default"}, destinationSecret)).To(Succeed())
			Expect(destinationSecret.Data).To(Equal(map[string][]byte{"key": []byte("value")}))
		})
	})

	AfterEach(func() {
		By("tearing down the test environment")
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())		// Clean up resources
	})

})

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}
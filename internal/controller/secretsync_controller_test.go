package controller_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	syncv1 "secret-sync-operator/api/v1"
	"secret-sync-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	testEnv   *envtest.Environment
	cfg       *rest.Config
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	ctx, cancel = context.WithCancel(context.Background())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		MetricsBindAddress: "0", // Disable metrics for tests
	})
	Expect(err).ToNot(HaveOccurred())

	reconciler := &controllers.SecretSyncReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}
	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	k8sClient = k8sManager.GetClient()

	go func() {
		defer GinkgoRecover()
		err := k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Controller", func() {
	Context("When reconciling SecretSync", func() {
		It("should reconcile secrets", func() {
			// Mock a SecretSync object
			secretSync := &syncv1.SecretSync{
				ObjectMeta: metav1.ObjectMeta{Name: "test-secretsync"},
				Spec: syncv1.SecretSyncSpec{
					Secrets: []string{"secret1", "secret2"},
				},
			}

			// Create the SecretSync object
			err := k8sClient.Create(ctx, secretSync)
			Expect(err).ToNot(HaveOccurred())

			// Mock a source secret
			sourceSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "secret1", Namespace: "default"},
				Data:       map[string][]byte{"key": []byte("value")},
			}

			// Create the source secret
			err = k8sClient.Create(ctx, sourceSecret)
			Expect(err).ToNot(HaveOccurred())

			// Wait for reconciliation
			Eventually(func() error {
				err := k8sClient.Get(ctx, client.ObjectKey{Name: "secret1", Namespace: "default"}, &corev1.Secret{})
				return err
			}).Should(Succeed())

			// Verify the destination secret is created
			destinationSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, client.ObjectKey{Name: "secret1", Namespace: "test"}, destinationSecret)
			Expect(err).ToNot(HaveOccurred())
			Expect(destinationSecret.Data).To(Equal(sourceSecret.Data))

			// Clean up
			err = k8sClient.Delete(ctx, secretSync)
			Expect(err).ToNot(HaveOccurred())
			err = k8sClient.Delete(ctx, sourceSecret)
			Expect(err).ToNot(HaveOccurred())
			err = k8sClient.Delete(ctx, destinationSecret)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

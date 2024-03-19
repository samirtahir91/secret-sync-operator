package controller

import (
    "context"
    "fmt"
    "path/filepath"
    "runtime"
    "testing"

    ctrl "sigs.k8s.io/controller-runtime"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "k8s.io/client-go/kubernetes/scheme"
    "k8s.io/client-go/rest"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/envtest"
    logf "sigs.k8s.io/controller-runtime/pkg/log"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"

	syncv1 "secret-sync-operator/api/v1"
    //+kubebuilder:scaffold:imports
)

var (
    cfg       *rest.Config
    k8sClient client.Client // You'll be using this client in your tests.
    testEnv   *envtest.Environment
    ctx       context.Context
    cancel    context.CancelFunc
)

func TestControllers(t *testing.T) {
    RegisterFailHandler(Fail)

    RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
    logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

    ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
    testEnv = &envtest.Environment{
        CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
        ErrorIfCRDPathMissing: true,

        // The BinaryAssetsDirectory is only required if you want to run the tests directly
        // without call the makefile target test. If not informed it will look for the
        // default path defined in controller-runtime which is /usr/local/kubebuilder/.
        // Note that you must have the required binaries setup under the bin directory to perform
        // the tests directly. When we run make test it will be setup and used automatically.
        BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
            fmt.Sprintf("1.29.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
    }

    var err error
    // cfg is defined in this file globally.
    cfg, err = testEnv.Start()
    Expect(err).NotTo(HaveOccurred())
    Expect(cfg).NotTo(BeNil())

    err = syncv1.AddToScheme(scheme.Scheme)
    Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme
    k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
    Expect(err).NotTo(HaveOccurred())
    Expect(k8sClient).NotTo(BeNil())

    k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
        Scheme: scheme.Scheme,
    })
    Expect(err).ToNot(HaveOccurred())

    err = (&SecretSyncReconciler{
        Client: k8sManager.GetClient(),
        Scheme: k8sManager.GetScheme(),
    }).SetupWithManager(k8sManager)
    Expect(err).ToNot(HaveOccurred())

    go func() {
        defer GinkgoRecover()
        err = k8sManager.Start(ctx)
        Expect(err).ToNot(HaveOccurred(), "failed to run manager")
    }()

})

var _ = AfterSuite(func() {
    cancel()
    By("tearing down the test environment")
    err := testEnv.Stop()
    Expect(err).NotTo(HaveOccurred())
})

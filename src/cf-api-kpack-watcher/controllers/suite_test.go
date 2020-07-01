/*


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

package controllers

import (
	"path/filepath"
	"testing"

	"code.cloudfoundry.org/capi-k8s-release/src/cf-api-kpack-watcher/cf"
	"code.cloudfoundry.org/capi-k8s-release/src/cf-api-kpack-watcher/cf/cffakes"
	"code.cloudfoundry.org/capi-k8s-release/src/cf-api-kpack-watcher/image_registry/image_registryfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	buildpivotaliov1alpha1 "github.com/pivotal/kpack/pkg/client/clientset/versioned/scheme"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sManager ctrl.Manager
var k8sClient client.Client
var testEnv *envtest.Environment

var (
	mockRestClient         *cffakes.FakeRest
	mockUAAClient          *cffakes.FakeTokenFetcher
	mockImageConfigFetcher *image_registryfakes.FakeImageConfigFetcher
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	// TODO: this is garbage, clean it up pls pls pls for the love of god
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	// TODO: is this generic scheme needed?
	err = scheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = buildpivotaliov1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sManager, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	// initialize mocks
	mockRestClient = &cffakes.FakeRest{}
	mockUAAClient = &cffakes.FakeTokenFetcher{}
	mockImageConfigFetcher = &image_registryfakes.FakeImageConfigFetcher{}

	// start controller with manager
	// TODO: refactor to remove mocks since this is an integration test
	err = (&BuildReconciler{
		Client: k8sManager.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Build"),
		Scheme: k8sManager.GetScheme(),
		CFClient: cf.NewClient(
			"https://cf.api",
			mockRestClient,
			mockUAAClient,
		),
		ImageConfigFetcher: mockImageConfigFetcher,
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		Expect(err).NotTo(HaveOccurred())
	}()

	k8sClient = k8sManager.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	// gexec.KillAndWait(5 * time.Second)
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
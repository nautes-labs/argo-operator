// Copyright 2023 Nautes Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	argocd "github.com/nautes-labs/argo-operator/pkg/argocd"
	secret "github.com/nautes-labs/argo-operator/pkg/secret"
	utilPort "github.com/nautes-labs/argo-operator/util/port"
	resourcev1alpha1 "github.com/nautes-labs/pkg/api/v1alpha1"
	zaplog "github.com/nautes-labs/pkg/pkg/log/zap"

	nautesconfigs "github.com/nautes-labs/pkg/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	//+kubebuilder:scaffold:imports
)

var (
	testEnv        *envtest.Environment
	k8sClient      client.Client
	gomockCtl      *gomock.Controller
	testKubeconfig *rest.Config
	fakeCtl        *fakeController
	nautesConfig   = &nautesconfigs.Config{
		Git: nautesconfigs.GitRepo{
			Addr:    "https://gitlab.bluzin.io",
			GitType: "gitlab",
		},
		Secret: nautesconfigs.SecretRepo{
			RepoType: "vault",
			Vault: nautesconfigs.Vault{
				Addr:      "https://vault.bluzin.io:8200",
				MountPath: "deployment2-runtime",
				ProxyAddr: "https://vault.bluzin.io:8000",
				Token:     "hvs.UseDJnYBNykFeGWlh7YAnzjC",
			},
		},
	}
	secretData = "apiVersion: v1\nclusters:\n- cluster:\n    server: https://127.0.0.1:6443\n    insecure-skip-tls-verify: true\n  name: local\ncontexts:\n- context:\n    cluster: local\n    namespace: default\n    user: user\n  name: pipeline1-runtime\ncurrent-context: pipeline1-runtime\nkind: Config\npreferences: {}\nusers:\n- name: user\n  user:\n    client-certificate-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJrRENDQVRlZ0F3SUJBZ0lJS005ZXhoLzlncXd3Q2dZSUtvWkl6ajBFQXdJd0l6RWhNQjhHQTFVRUF3d1kKYXpOekxXTnNhV1Z1ZEMxallVQXhOalUyTXprNU5UYzFNQjRYRFRJeU1EWXlPREEyTlRrek5Wb1hEVEl6TURZeQpPREEyTlRrek5Wb3dNREVYTUJVR0ExVUVDaE1PYzNsemRHVnRPbTFoYzNSbGNuTXhGVEFUQmdOVkJBTVRESE41CmMzUmxiVHBoWkcxcGJqQlpNQk1HQnlxR1NNNDlBZ0VHQ0NxR1NNNDlBd0VIQTBJQUJMNEJ3TDFBMVhYaStTcm8KYlJRSklFS3FhMlUycEZKV2NUcWJ1SmtKdVp5ZDBrdFBUOHhscmVoQ1phUzhRajFHdVVvWmNBM1A5eE12dVBWTgovTUx6UGVXalNEQkdNQTRHQTFVZER3RUIvd1FFQXdJRm9EQVRCZ05WSFNVRUREQUtCZ2dyQmdFRkJRY0RBakFmCkJnTlZIU01FR0RBV2dCU3VTbUJQeFRmdHVoSXBHczBpSjRSd0MyVVY5VEFLQmdncWhrak9QUVFEQWdOSEFEQkUKQWlCckF5S2lNaWZUTUs3MThpd0hkVGZxUFI1TU96QW9OTjVvajZDak9uSzZtUUlnZG5HNFJQVjJtYTQ4R2FmQQovVXJUeEV3R0g3WVcwQ3VvUzZjL0tjWU5RbjA9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0KLS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUJkekNDQVIyZ0F3SUJBZ0lCQURBS0JnZ3Foa2pPUFFRREFqQWpNU0V3SHdZRFZRUUREQmhyTTNNdFkyeHAKWlc1MExXTmhRREUyTlRZek9UazFOelV3SGhjTk1qSXdOakk0TURZMU9UTTFXaGNOTXpJd05qSTFNRFkxT1RNMQpXakFqTVNFd0h3WURWUVFEREJock0zTXRZMnhwWlc1MExXTmhRREUyTlRZek9UazFOelV3V1RBVEJnY3Foa2pPClBRSUJCZ2dxaGtqT1BRTUJCd05DQUFROFdnVHJ6cDNkZVNlMFlVTTNIamhQZGU2VWRoNGN2L0h2RStTci9BcDQKVGxnc2srSVcwSG5jSHoyVXQ2c0lXMkRRenNVMnJyMGZXZ0pmVlNZendoS0NvMEl3UURBT0JnTlZIUThCQWY4RQpCQU1DQXFRd0R3WURWUjBUQVFIL0JBVXdBd0VCL3pBZEJnTlZIUTRFRmdRVXJrcGdUOFUzN2JvU0tSck5JaWVFCmNBdGxGZlV3Q2dZSUtvWkl6ajBFQXdJRFNBQXdSUUlnSGJ3aFdLQ2Jxa3IxcEFRdk04bGtrNC9Pc0hiWVZZTEMKVVo5Q1lmVWx1bE1DSVFEcW9TVVBkb3F0cUpTRnZ2bnkxQjBVeXBkRWRzemMzczBuSk5PekhEdU12dz09Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K\n    client-key-data: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1IY0NBUUVFSVBKZmQvVEx0MnZSczJEOHlWVklTV0xSL3NHVm1ZbjRvNjFMVkNMb29VZUtvQW9HQ0NxR1NNNDkKQXdFSG9VUURRZ0FFdmdIQXZVRFZkZUw1S3VodEZBa2dRcXByWlRha1VsWnhPcHU0bVFtNW5KM1NTMDlQekdXdAo2RUpscEx4Q1BVYTVTaGx3RGMvM0V5KzQ5VTM4d3ZNOTVRPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=\n"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

type fakeController struct {
	k8sManager manager.Manager
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewFakeController() *fakeController {
	ctrl.SetLogger(zaplog.New())

	var port int
	port, _ = utilPort.GetAvaliablePort()
	ok := utilPort.IsPortAvaliable(port)
	if !ok {
		port = 8000
	}

	k8sManager, err := ctrl.NewManager(testKubeconfig, ctrl.Options{Scheme: scheme.Scheme, MetricsBindAddress: fmt.Sprintf(":%d", port)})
	Expect(err).ToNot(HaveOccurred())
	ctx, cancel := context.WithCancel(context.TODO())

	return &fakeController{
		k8sManager: k8sManager,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (f *fakeController) startCluster(argocd *argocd.ArgocdClient, secret secret.SecretOperator, config *nautesconfigs.Config) {
	if f.k8sManager != nil {
		(&ClusterReconciler{
			Scheme: f.k8sManager.GetScheme(),
			Client: f.k8sManager.GetClient(),
			Argocd: argocd,
			Secret: secret,
			Log:    ctrl.Log.WithName("cluster controller test log"),
		}).SetupWithManager(f.k8sManager)

		go func() {
			defer GinkgoRecover()
			err := f.k8sManager.Start(f.ctx)
			Expect(err).ToNot(HaveOccurred())
		}()
	}
}

func (f *fakeController) close() {
	f.cancel()
	time.Sleep(3 * time.Second)
}

func (f *fakeController) GetClient() client.Client {
	return f.k8sManager.GetClient()
}

var _ = BeforeSuite(func() {
	_ = ctrl.Log.WithName("BeforeSuite log")

	var use = false
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:        []string{filepath.Join("..", "..", "config", "crd", "bases")},
		UseExistingCluster:       &use,
		AttachControlPlaneOutput: false,
	}

	var err error
	testKubeconfig, err = testEnv.Start()
	if err != nil {
		os.Exit(1)
	}

	err = resourcev1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme
	gomockCtl = gomock.NewController(GinkgoT())

	client, err := client.New(testKubeconfig, client.Options{})
	Expect(err).ShouldNot(HaveOccurred())

	err = createDefaultNamespace(client)
	Expect(err).NotTo(HaveOccurred())

	err = createNautesConfigs(client)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	defer testEnv.Stop()
})

func createDefaultNamespace(c client.Client) error {
	var defaultNamespace = "nautes"

	ns := corev1.Namespace{}
	err := c.Get(context.Background(), client.ObjectKey{Name: defaultNamespace}, &ns)
	if err != nil {
		if errors.IsNotFound(err) {
			ns.Name = defaultNamespace
			err = c.Create(context.Background(), &ns)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func createNautesConfigs(c client.Client) error {
	namespace := "nautes"
	name := "nautes-configs"
	err := c.Get(context.Background(), client.ObjectKey{Name: name, Namespace: namespace}, &corev1.ConfigMap{})
	if err != nil {
		if errors.IsNotFound(err) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}
			if err := c.Create(context.Background(), cm); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

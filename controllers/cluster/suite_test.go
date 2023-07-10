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

	"github.com/go-logr/logr"
	argocd "github.com/nautes-labs/argo-operator/pkg/argocd"
	secret "github.com/nautes-labs/argo-operator/pkg/secret"
	utilPort "github.com/nautes-labs/argo-operator/util/port"
	zaplog "github.com/nautes-labs/pkg/pkg/log/zap"

	nautesconfigs "github.com/nautes-labs/pkg/pkg/nautesconfigs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	//+kubebuilder:scaffold:imports
)

var (
	testEnv *envtest.Environment
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

type fakeController struct {
	k8sManager manager.Manager
	ctx        context.Context
	cancel     context.CancelFunc
	waitClose  chan bool
	kubeconfig *rest.Config
}

func NewFakeController(kubeconfig *rest.Config) *fakeController {
	ctrl.SetLogger(zaplog.New())

	var port int
	port, _ = utilPort.GetAvaliablePort()
	ok := utilPort.IsPortAvaliable(port)
	if !ok {
		port = 8000
	}

	k8sManager, err := ctrl.NewManager(kubeconfig, ctrl.Options{Scheme: scheme.Scheme, MetricsBindAddress: fmt.Sprintf(":%d", port)})
	Expect(err).ToNot(HaveOccurred())
	ctx, cancel := context.WithCancel(context.Background())

	return &fakeController{
		k8sManager: k8sManager,
		ctx:        ctx,
		cancel:     cancel,
		waitClose:  make(chan bool),
		kubeconfig: kubeconfig,
	}
}

func NewReconciler(scheme *runtime.Scheme, k8sClient client.Client, argocd *argocd.ArgocdClient, secret secret.SecretOperator, log logr.Logger) *ClusterReconciler {
	return &ClusterReconciler{
		Scheme: scheme,
		Client: k8sClient,
		Argocd: argocd,
		Secret: secret,
		Log:    log,
	}
}

func initialEnvTest() (*rest.Config, error) {
	use := false
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:  []string{filepath.Join("..", "..", "config", "crd", "bases")},
		UseExistingCluster: &use,
	}
	kubeconfig, err := testEnv.Start()
	if err != nil {
		return nil, err
	}

	return kubeconfig, nil
}

func stopEnvTest() error {
	return testEnv.Stop()
}

func (f *fakeController) startCluster(argocd *argocd.ArgocdClient, secret secret.SecretOperator, config *nautesconfigs.Config) {
	client, err := client.New(f.kubeconfig, client.Options{})
	Expect(err).ShouldNot(HaveOccurred())

	err = createDefaultNamespace(client)
	Expect(err).NotTo(HaveOccurred())

	err = createNautesConfigs(client)
	Expect(err).NotTo(HaveOccurred())

	scheme := f.k8sManager.GetScheme()
	k8sClient := f.k8sManager.GetClient()
	log := ctrl.Log.WithName("cluster controller test log")
	reconciler := NewReconciler(scheme, k8sClient, argocd, secret, log)
	reconciler.SetupWithManager(f.k8sManager)

	go func() {
		defer GinkgoRecover()

		defer func() {
			f.waitClose <- true
		}()

		err := f.k8sManager.Start(f.ctx)
		Expect(err).ShouldNot(HaveOccurred())
	}()
}

func (f *fakeController) close() {
	f.cancel()
	message := <-f.waitClose
	fmt.Println("Have closed Controller", message)

	err := stopEnvTest()
	if err != nil {
		os.Exit(1)
	}
}

func (f *fakeController) GetClient() client.Client {
	return f.k8sManager.GetClient()
}

func createDefaultNamespace(c client.Client) error {
	ns := corev1.Namespace{}
	err := c.Get(context.Background(), client.ObjectKey{Name: DefaultNamespace}, &ns)
	if err != nil {
		if errors.IsNotFound(err) {
			ns.Name = DefaultNamespace
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
	namespace := DefaultNamespace
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

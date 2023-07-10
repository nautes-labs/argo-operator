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
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	argocd "github.com/nautes-labs/argo-operator/pkg/argocd"
	pkgsecret "github.com/nautes-labs/argo-operator/pkg/secret"
	resourcev1alpha1 "github.com/nautes-labs/pkg/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nautesconfigs "github.com/nautes-labs/pkg/pkg/nautesconfigs"
)

var (
	errSyncCluster     = errors.New("cluster addition failed")
	errDeleteCluster   = errors.New("cluster deletion failed")
	errNotFoundCluster = errors.New("cluster is not found")
	errGetSecret       = errors.New("secret path is error")
)

var _ = Describe("Cluster controller test cases", func() {
	const (
		timeout         = time.Second * 60
		interval        = time.Second * 3
		CLUSTER_ADDRESS = "https://kubernetes.default.svc"
	)

	var (
		k8sClient    client.Client
		gomockCtl    *gomock.Controller
		fakeCtl      *fakeController
		nautesConfig = &nautesconfigs.Config{
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

	var (
		clusterReponse = &argocd.ClusterResponse{
			Name:   "argo-operator-test-vcluster",
			Server: CLUSTER_ADDRESS,
			ConnectionState: argocd.ConnectionState{
				Status: "Successful",
			},
		}
		clusterFailReponse = &argocd.ClusterResponse{
			Name:   "argo-operator-test-vcluster",
			Server: CLUSTER_ADDRESS,
			ConnectionState: argocd.ConnectionState{
				Status: "Fail",
			},
		}
	)

	BeforeEach(func() {
		err := resourcev1alpha1.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())
		gomockCtl = gomock.NewController(GinkgoT())
	})

	It("successfully create cluster to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(CLUSTER_ADDRESS, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetToken(gomock.Any()).Return("tokeeeen", nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expected resource created successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			if err == nil {
				err = k8sClient.Delete(context.Background(), cluster)
				if err == nil {
					return true
				}
			}

			return false
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("failed to update cluster to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		first := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errors.New("failed to get cluster info"))
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterFailReponse, nil).After(first).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().UpdateCluster(gomock.Any()).Return(errors.New("failed to update cluster")).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(CLUSTER_ADDRESS, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetToken(gomock.Any()).Return("tokeeeen", nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expected resource created successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant2",
			Usage:       "worker",
		}
		toUpdate := toCreate.DeepCopy()
		toUpdate.Spec = updateSpec

		By("Expected resource update successfully")
		k8sClient.Get(context.Background(), key, toUpdate)
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "False"
			}

			return false
		}, timeout*2, interval*2).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			if err == nil {
				err = k8sClient.Delete(context.Background(), cluster)
				if err == nil {
					return true
				}
			}

			return false
		}, timeout, interval).Should(BeTrue())
	})

	It("cluster validate error", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, nil, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "physical",
			ClusterKind: "kubernetes",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expected resource created successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			conditions := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			if len(conditions) > 0 {
				return conditions[0].Status == "False"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			if err == nil {
				err = k8sClient.Delete(context.Background(), cluster)
				if err == nil {
					return true
				}
			}

			return false
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("when secret has change create cluster to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		secondGetCluster := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).AnyTimes().Times(2)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).AnyTimes().After(secondGetCluster)
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return("https://kubernetes.default.svc1", nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		firstGetSecret := secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil)
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 2, Data: secretData}, nil).AnyTimes().After(firstGetSecret)

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 6)

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant2",
			Usage:       "worker",
		}

		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			if cluster.Status.Sync2ArgoStatus != nil {
				return cluster.Status.Sync2ArgoStatus.SecretID == "2"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			condition := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			return len(condition) == 1
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			if err == nil {
				err = k8sClient.Delete(context.Background(), cluster)
				if err == nil {
					return true
				}
			}

			return false
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("update resource successfully update cluster to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		firstGetCluster := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterFailReponse, nil).AnyTimes().After(firstGetCluster)
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().UpdateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return("https://kubernetes.default.svc1", nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			condition := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			return len(condition) == 1
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   "https://kubernetes.default.svc1",
			ClusterKind: "kubernetes",
			ClusterType: "virtual",
			HostCluster: "tenant2",
			Usage:       "worker",
		}

		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			condition := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			return len(condition) == 1
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			if err == nil {
				err = k8sClient.Delete(context.Background(), cluster)
				if err == nil {
					return true
				}
			}

			return false
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("update resource successfully create cluster to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return("https://kubernetes.default.svc1", nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			condition := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			return len(condition) == 1
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   "https://kubernetes.default.svc1",
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant2",
			Usage:       "worker",
		}

		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			condition := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			return len(condition) == 1
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			if err == nil {
				err = k8sClient.Delete(context.Background(), cluster)
				if err == nil {
					return true
				}
			}

			return false
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("failed to save cluster to argocd", func() {
		time.Sleep(time.Second * 3)

		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(errSyncCluster).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(CLUSTER_ADDRESS, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetToken(gomock.Any()).Return("tokeeeen", nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		createdCluster := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expected conditions of resource status is nil")
		err = k8sClient.Create(context.Background(), createdCluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			if err == nil && len(cluster.Status.Conditions) > 0 {
				return cluster.Status.Conditions[0].Status == "False"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			if err == nil {
				err = k8sClient.Delete(context.Background(), cluster)
				if err == nil {
					return true
				}
			}

			return false
		}, timeout, interval).Should(BeTrue())

		fakeCtl.close()
	})

	It("unable to get secret data", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		firstGetCluster := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).AnyTimes()
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).AnyTimes().After(firstGetCluster)
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(CLUSTER_ADDRESS, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetToken(gomock.Any()).Return("tokeeeen", nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{}, errGetSecret).AnyTimes()

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		createdCluster := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expected conditions of resource status is nil")
		err = k8sClient.Create(context.Background(), createdCluster)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Status.Sync2ArgoStatus == nil
		}, timeout, interval).Should(BeTrue())

		defer fakeCtl.close()
	})

	It("failed to update cluster", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		firstGetCluster := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).AnyTimes()
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).AnyTimes().After(firstGetCluster)
		firstCreateCluster := argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(errSyncCluster).After(firstCreateCluster).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(CLUSTER_ADDRESS, errDeleteCluster).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		createdCluster := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), createdCluster)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			condition := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			return len(condition) == 1
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   "https://kubernetes.default.svc2",
			ClusterKind: "kubernetes",
			ClusterType: "virtual",
			HostCluster: "tenant2",
			Usage:       "worker",
		}

		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			condition := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			return len(condition) != 0
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			cluster := &resourcev1alpha1.Cluster{}
			err := k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			return k8sClient.Delete(context.Background(), cluster)
		}, timeout, interval).Should(Succeed())

		By("Expected resource deletion finished")
		Eventually(func() error {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("update resource failed to save cluster to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		firstGetCluster := argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(nil, errNotFoundCluster).AnyTimes()
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterReponse, nil).AnyTimes().After(firstGetCluster)
		firstCreateCluster := argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(nil).AnyTimes()
		argocdCluster.EXPECT().CreateCluster(gomock.Any()).Return(errSyncCluster).After(firstCreateCluster).AnyTimes()
		argocdCluster.EXPECT().DeleteCluster(gomock.Any()).Return(CLUSTER_ADDRESS, nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:    argocdAuth,
			ArgocdCluster: argocdCluster,
		}

		secret := pkgsecret.NewMockSecretOperator(gomockCtl)
		secret.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		secret.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		secret.EXPECT().GetSecret(gomock.Any()).Return(&pkgsecret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		cfg, err := initialEnvTest()
		Expect(err).ShouldNot(HaveOccurred())
		fakeCtl = NewFakeController(cfg)
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCluster(argocd, secret, nautesConfig)

		// Resource
		spec := resourcev1alpha1.ClusterSpec{
			ApiServer:   CLUSTER_ADDRESS,
			ClusterKind: "kubernetes",
			ClusterType: "virtual",
			HostCluster: "tenant1",
			Usage:       "worker",
		}

		var resourceName = spliceResourceName("cluster")
		key := types.NamespacedName{
			Namespace: DefaultNamespace,
			Name:      resourceName,
		}

		createdCluster := &resourcev1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expecting resource creation successfully")
		err = k8sClient.Create(context.Background(), createdCluster)
		Expect(err).ShouldNot(HaveOccurred())

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			condition := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			return len(condition) == 1
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.ClusterSpec{
			ApiServer:   "https://kubernetes.default.svc2",
			ClusterType: "virtual",
			ClusterKind: "kubernetes",
			HostCluster: "tenant2",
			Usage:       "worker",
		}
		toUpdate := &resourcev1alpha1.Cluster{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec

		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.ClusterSpec {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			return cluster.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			condition := cluster.Status.GetConditions(map[string]bool{ClusterConditionType: true})
			return len(condition) == 1
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			cluster := &resourcev1alpha1.Cluster{}
			err := k8sClient.Get(context.Background(), key, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			return k8sClient.Delete(context.Background(), cluster)
		}, timeout, interval).Should(Succeed())

		By("Expected resource deletion finished")
		Eventually(func() error {
			cluster := &resourcev1alpha1.Cluster{}
			err = k8sClient.Get(context.Background(), key, cluster)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})
})

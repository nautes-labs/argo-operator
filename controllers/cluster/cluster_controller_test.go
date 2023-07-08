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
)

var (
	errSyncCluster     = errors.New("cluster addition failed")
	errDeleteCluster   = errors.New("cluster deletion failed")
	errNotFoundCluster = errors.New("cluster is not found")
	errGetSecret       = errors.New("secret path is error")
)

var _ = Describe("Cluster controller test cases", func() {
	const timeout = time.Second * 20
	const interval = time.Second * 8
	const CLUSTER_ADDRESS = "https://kubernetes.default.svc"

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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), toCreate)
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

	It("successfully update cluster to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCluster := argocd.NewMockClusterOperation(gomockCtl)
		argocdCluster.EXPECT().GetClusterInfo(gomock.Any()).Return(clusterFailReponse, nil).AnyTimes()
		argocdCluster.EXPECT().UpdateCluster(gomock.Any()).Return(nil).AnyTimes()
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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), toCreate)
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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), toCreate)
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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 8)

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			cluster := &resourcev1alpha1.Cluster{}
			k8sClient.Get(context.Background(), key, cluster)
			if len(cluster.Status.Conditions) > 0 {
				return cluster.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), toCreate)
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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), toCreate)
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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), createdCluster)
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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), createdCluster)
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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), createdCluster)
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
		fakeCtl = NewFakeController()
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
			Namespace: "nautes",
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
		err := k8sClient.Create(context.Background(), createdCluster)
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

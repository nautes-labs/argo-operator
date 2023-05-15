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

package coderepo

import (
	"context"
	"errors"
	"time"

	"github.com/golang/mock/gomock"
	argocd "github.com/nautes-labs/argo-operator/pkg/argocd"
	secret "github.com/nautes-labs/argo-operator/pkg/secret"
	resourcev1alpha1 "github.com/nautes-labs/pkg/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

var (
	errNotFoundCodeRepo = errors.New("coderepo is not found")
	errCreateRepository = errors.New("failed to create repository")
	errDeleteRepository = errors.New("failed to delete repository")
	errGetSecret        = errors.New("failed to get secret")
)

var _ = Describe("CodeRepo controller test cases", func() {
	const timeout = time.Second * 20
	const interval = time.Second * 5
	const CODEREPO_URL = "git@158.222.222.235:nautes/test.git"
	const CODEREPO_UPDATE_URL = "ssh://git@gitlab.com:2222/nautes-labs/test.git"

	var (
		codeRepoReponse = &argocd.CodeRepoResponse{
			Name: "argo-operator-test-vcluster",
			Repo: CODEREPO_URL,
			ConnectionState: argocd.ConnectionState{
				Status: "Successful",
			},
		}
		codeRepoFailReponse = &argocd.CodeRepoResponse{
			Name: "argo-operator-test-vcluster",
			Repo: CODEREPO_URL,
			ConnectionState: argocd.ConnectionState{
				Status: "Fail",
			},
		}
		errNotFound = &argocd.ErrorNotFound{Code: 404, Message: errNotFoundCodeRepo.Error()}
	)

	It("successfully create repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		firstGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoReponse, nil).AnyTimes().After(firstGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().InitVault(gomock.Any()).Return(nil)
		sc.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		By("Expected resource created successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 6)

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err = k8sClient.Get(context.Background(), key, codeRepo)
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return k8sClient.Delete(context.Background(), codeRepo)
		}, timeout, interval).Should(BeNil())

		fakeCtl.close()
	})

	It("successfully update repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()
		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoFailReponse, nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().InitVault(gomock.Any()).Return(nil)
		sc.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
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

		time.Sleep(time.Second * 10)

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err = k8sClient.Get(context.Background(), key, codeRepo)
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			err := k8sClient.Delete(context.Background(), codeRepo)
			return err
		}, timeout, interval).Should(BeNil())

		fakeCtl.close()
	})

	It("is resource validate error", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound).AnyTimes()
		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, nil, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
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

		time.Sleep(time.Second * 5)

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err = k8sClient.Get(context.Background(), key, codeRepo)
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "False"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			err := k8sClient.Delete(context.Background(), codeRepo)
			return err
		}, timeout, interval).Should(BeNil())

		By("Expected resource deletion finished")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			return err
		}, timeout, interval).ShouldNot(Succeed())

		fakeCtl.close()
	})

	It("updates repository to argocd when secret has change", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		firstGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoFailReponse, nil).AnyTimes().After(firstGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		firstGetSecret := sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil)
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 2, Data: secretData}, nil).AnyTimes().After(firstGetSecret)

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
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

		time.Sleep(time.Second * 5)
		By("Expecting cluster added argocd")
		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   true,
			DeploymentRuntime: false,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"TagPushEvents"},
			},
		}

		toUpdate := &resourcev1alpha1.CodeRepo{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return codeRepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if codeRepo.Status.Sync2ArgoStatus != nil {
				return codeRepo.Status.Sync2ArgoStatus.SecretID == "2"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			c := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, c)
			Expect(err).ShouldNot(HaveOccurred())
			err = k8sClient.Delete(context.Background(), c)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("creates repository to argocd when secret has change", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		secondGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoFailReponse, nil).AnyTimes().After(secondGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		firstGetSecret := sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil)
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 2, Data: secretData}, nil).AnyTimes().After(firstGetSecret)

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
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

		By("Expecting resource added argocd")
		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   true,
			DeploymentRuntime: false,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"TagPushEvents"},
			},
		}

		toUpdate := &resourcev1alpha1.CodeRepo{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return codeRepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if codeRepo.Status.Sync2ArgoStatus != nil {
				return codeRepo.Status.Sync2ArgoStatus.SecretID == "2"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion succeeded")
		Eventually(func() error {
			c := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, c)
			Expect(err).ShouldNot(HaveOccurred())
			err = k8sClient.Delete(context.Background(), c)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("modify resource successfully and update repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		firstGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoFailReponse, nil).AnyTimes().After(firstGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		By("Expecting resource creation successfully")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		time.Sleep(time.Second * 5)

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			condition := codeRepo.Status.GetConditions(map[string]bool{CodeRepoConditionType: true})
			return len(condition) > 0
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_UPDATE_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			toUpdate := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, toUpdate)
			toUpdate.Spec = updateSpec
			err = k8sClient.Update(context.Background(), toUpdate)
			return err
		})
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return codeRepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			Expect(err).ShouldNot(HaveOccurred())

			condition := codeRepo.Status.GetConditions(map[string]bool{CodeRepoConditionType: true})

			return len(condition) > 0
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			c := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, c)
			Expect(err).ShouldNot(HaveOccurred())
			err = k8sClient.Delete(context.Background(), c)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("modify resource successfully and create repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound).AnyTimes()
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
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
		time.Sleep(time.Second * 5)

		By("Expecting cluster added argocd")
		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the coderepo is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_UPDATE_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			toUpdate := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, toUpdate)
			toUpdate.Spec = updateSpec
			err = k8sClient.Update(context.Background(), toUpdate)
			return err
		})
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return codeRepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			Expect(err).ShouldNot(HaveOccurred())

			condition := codeRepo.Status.GetConditions(map[string]bool{CodeRepoConditionType: true})

			return len(condition) > 0
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			c := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, c)
			Expect(err).ShouldNot(HaveOccurred())
			err = k8sClient.Delete(context.Background(), c)
			return err
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("failed to create repository to argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound).AnyTimes()
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(errCreateRepository).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		// Create
		By("Expected conditions of resource status is nil")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "False"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Delete
		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return k8sClient.Delete(context.Background(), codeRepo)
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("failed to get secret data", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound).AnyTimes()
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(errCreateRepository).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(nil).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{}, errGetSecret).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
			ObjectMeta: metav1.ObjectMeta{
				Name:      key.Name,
				Namespace: key.Namespace,
			},
			Spec: spec,
		}

		By("Expected conditions of resource status is nil")
		err := k8sClient.Create(context.Background(), toCreate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			if len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "False"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion succeeded")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, codeRepo)
			return k8sClient.Delete(context.Background(), codeRepo)
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})

	It("failed to delete repository of argocd", func() {
		argocdAuth := argocd.NewMockAuthOperation(gomockCtl)
		argocdAuth.EXPECT().GetPassword().Return("pass", nil).AnyTimes()
		argocdAuth.EXPECT().Login().Return(nil).AnyTimes()

		argocdCodeRepo := argocd.NewMockCoderepoOperation(gomockCtl)
		firstGetCodeRepo := argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(nil, errNotFound)
		argocdCodeRepo.EXPECT().GetRepositoryInfo(gomock.Any()).Return(codeRepoReponse, nil).AnyTimes().After(firstGetCodeRepo)
		argocdCodeRepo.EXPECT().CreateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().UpdateRepository(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		argocdCodeRepo.EXPECT().DeleteRepository(gomock.Any()).Return(errDeleteRepository).AnyTimes()

		argocd := &argocd.ArgocdClient{
			ArgocdAuth:     argocdAuth,
			ArgocdCodeRepo: argocdCodeRepo,
		}

		sc := secret.NewMockSecretOperator(gomockCtl)
		sc.EXPECT().InitVault(gomock.Any()).Return(nil).AnyTimes()
		sc.EXPECT().GetToken(gomock.Any()).Return("token", nil).AnyTimes()
		sc.EXPECT().GetSecret(gomock.Any()).Return(&secret.SecretData{ID: 1, Data: secretData}, nil).AnyTimes()

		// Initial fakeCtl controller instance
		fakeCtl = NewFakeController()
		k8sClient = fakeCtl.GetClient()
		fakeCtl.startCodeRepo(argocd, sc, nautesConfig)

		// Resource
		spec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents", "TagPushEvents"},
			},
		}

		var resourceName = generateResourceName("coderepo")
		key := types.NamespacedName{
			Namespace: "nautes",
			Name:      resourceName,
		}

		toCreate := &resourcev1alpha1.CodeRepo{
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
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}

			return false
		}, timeout, interval).Should(BeTrue())

		// Update
		By("Expected the cluster is to be updated")
		updateSpec := resourcev1alpha1.CodeRepoSpec{
			URL:               CODEREPO_URL,
			PipelineRuntime:   false,
			DeploymentRuntime: true,
			Webhook: &resourcev1alpha1.Webhook{
				Events: []string{"PushEvents"},
			},
		}

		toUpdate := &resourcev1alpha1.CodeRepo{}
		k8sClient.Get(context.Background(), key, toUpdate)
		toUpdate.Spec = updateSpec
		err = k8sClient.Update(context.Background(), toUpdate)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() resourcev1alpha1.CodeRepoSpec {
			coderepo := &resourcev1alpha1.CodeRepo{}
			k8sClient.Get(context.Background(), key, coderepo)
			return coderepo.Spec
		}, timeout, interval).Should(Equal(updateSpec))

		Eventually(func() bool {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			Expect(err).ShouldNot(HaveOccurred())
			if err == nil && len(codeRepo.Status.Conditions) > 0 {
				return codeRepo.Status.Conditions[0].Status == "True"
			}
			return false
		}, timeout, interval).Should(BeTrue())

		By("Expected resource deletion failed")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			err := k8sClient.Get(context.Background(), key, codeRepo)
			if err != nil {
				return err
			}
			return k8sClient.Delete(context.Background(), codeRepo)
		}, timeout, interval).Should(Succeed())

		By("Expected resource deletion incomplete")
		Eventually(func() error {
			codeRepo := &resourcev1alpha1.CodeRepo{}
			return k8sClient.Get(context.Background(), key, codeRepo)
		}, timeout, interval).Should(Succeed())

		fakeCtl.close()
	})
})

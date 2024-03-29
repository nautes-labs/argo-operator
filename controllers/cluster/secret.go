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
	"strconv"

	secret "github.com/nautes-labs/argo-operator/pkg/secret"
	resourcev1alpha1 "github.com/nautes-labs/pkg/api/v1alpha1"
	nautesconfigs "github.com/nautes-labs/pkg/pkg/nautesconfigs"
	"k8s.io/kops/pkg/kubeconfig"
)

type SecretContent struct {
	ID         string
	Kubeconfig *kubeconfig.KubectlConfig
	PrivateKey string
}

// isSecretChange if secret id has been changed return true
func (r *ClusterReconciler) isSecretChange(cluster *resourcev1alpha1.Cluster, id string) bool {
	if cluster.Status.Sync2ArgoStatus != nil && cluster.Status.Sync2ArgoStatus.SecretID != "" {
		return id != cluster.Status.Sync2ArgoStatus.SecretID
	}

	return false
}

// Get stored key and vaule using vault secret
func (r *ClusterReconciler) getSecret(ctx context.Context, clusterName, namespace string, configs *nautesconfigs.Config) (*SecretContent, error) {
	secretPath := fmt.Sprintf("kubernetes/%s/%s/%s", clusterName, "default", "admin")
	secretsEngine := "cluster"
	secretsKey := "kubeconfig"

	secretOptions := secret.SecretOptions{
		SecretPath:   secretPath,
		SecretEngine: secretsEngine,
		SecretKey:    secretsKey,
	}

	secretConfig := &secret.SecretConfig{
		Namespace:  configs.Nautes.Namespace,
		SecretRepo: &configs.Secret,
	}

	if err := r.Secret.Init(secretConfig); err != nil {
		return nil, err
	}

	secret, err := r.Secret.GetSecret(secretOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret, err: %w", err)
	}

	kubeconfig, err := r.ConvertKubeconfig([]byte(secret.Data))
	if err != nil {
		return nil, err
	}

	return &SecretContent{
		ID:         strconv.Itoa(secret.ID),
		Kubeconfig: kubeconfig,
	}, nil
}

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

package secret

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	DefaultNamespace      = "nautes"
	DefaultServiceAccount = "argo-controller-manager"
	SecretsEngine         = "git"
	SecretsKey            = "deploykey"
	AuthRoleKey           = "Argo"
)

type VaultConfig struct {
	Addr         string
	CABundle     string
	MountPath    string
	Namespace    string
	OperatorName map[string]string
}

type VaultClient struct {
	client      *vault.Client
	VaultConfig *VaultConfig
}

func NewVaultClient() (SecretOperator, error) {
	return &VaultClient{}, nil
}

func (v *VaultClient) InitVault(config *VaultConfig) (*VaultClient, error) {
	httpClient, err := NewHttpClient(config.CABundle)
	if err != nil {
		return nil, err
	}

	token, err := v.GetToken(config.Namespace)
	if err != nil {
		return nil, err
	}

	kubernetesAuth, err := NewKubernetesAuth(config.MountPath, token, config.OperatorName)
	if err != nil {
		return nil, err
	}

	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = config.Addr
	vaultConfig.HttpClient = httpClient

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, err
	}
	v.client = client

	authInfo, err := client.Auth().Login(context.Background(), kubernetesAuth)
	if err != nil {
		return nil, fmt.Errorf("unable to log in with Kubernetes auth: %w", err)
	}

	if authInfo == nil {
		return nil, fmt.Errorf("no auth info was returned after login")
	}

	return v, nil
}

func (s *VaultClient) Logout(client *vault.Client) error {
	err := client.Auth().Token().RevokeSelf("")
	if err != nil {
		return err
	}
	return nil
}

func (v *VaultClient) GetSecret(secretOptions SecretOptions) (*SecretData, error) {
	defer v.Logout(v.client)

	secret, err := v.client.KVv2(secretOptions.SecretEngine).Get(context.Background(), secretOptions.SecretPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read secret: %w", err)
	}

	metadata, err := v.client.KVv2(secretOptions.SecretEngine).GetVersionsAsList(context.Background(), secretOptions.SecretPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read versions list: %w", err)
	}

	len := len(metadata)
	lastSecretMetadata := metadata[len-1]

	secretValue := secret.Data[secretOptions.SecretKey]
	if secretValue == nil {
		return nil, fmt.Errorf("secret data %s is not found", secretOptions.SecretKey)
	}

	return &SecretData{
		ID:   lastSecretMetadata.Version,
		Data: secretValue.(string),
	}, nil
}

func (v *VaultClient) RenewToken(increment int) error {
	renewedToken, err := v.client.Auth().Token().RenewSelf(increment)
	if err != nil {
		return err
	}

	v.client.SetToken(renewedToken.Auth.ClientToken)

	return nil
}

func (v *VaultClient) GetToken(namespace string) (string, error) {
	sa := &corev1.ServiceAccount{}
	saNamespaceName := types.NamespacedName{
		Namespace: namespace,
		Name:      DefaultServiceAccount,
	}

	client, err := NewKubernetesClient()
	if err != nil {
		return "", err
	}

	err = client.Get(context.Background(), saNamespaceName, sa)
	if err != nil {
		return "", err
	}

	secretName := sa.Secrets[0].Name
	secret := &corev1.Secret{}
	secretNamespaceName := types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}

	err = client.Get(context.Background(), secretNamespaceName, secret)
	if err != nil {
		return "", err
	}
	return string(secret.Data["token"]), nil
}

func NewKubernetesClient() (client.Client, error) {
	client, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func NewHttpClient(ca string) (*http.Client, error) {
	if ca == "" {
		return nil, fmt.Errorf("failed to get vault cert")
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM([]byte(ca))
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}, nil
}

func NewKubernetesAuth(mountPath, token string, roles map[string]string) (*auth.KubernetesAuth, error) {
	if mountPath == "" {
		return nil, fmt.Errorf("failed to get vault mount path")
	}

	role, ok := roles[AuthRoleKey]
	if !ok {
		return nil, fmt.Errorf("failed to get argo-operator role in nautes config")
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		role,
		auth.WithServiceAccountToken(token),
		auth.WithMountPath(mountPath),
	)

	if err != nil {
		return nil, fmt.Errorf("unable to initialize Kubernetes auth method: %w", err)
	}

	return k8sAuth, nil
}
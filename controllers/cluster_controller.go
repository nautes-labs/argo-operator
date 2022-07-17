/*
Copyright 2022.

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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"reflect"

	argov1alpha1 "argo-operator/api/v1alpha1"
	resourcev1alpha1 "argo-operator/api/v1alpha1"
	utilstring "argo-operator/util/string"

	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const (
	ARGOCDINITIALADMINSECRET = "argocd-initial-admin-secret"
	ARGOCDNAMESPACE          = "argocd"
	VAULTHOST                = "https://10.204.118.215:8200"
)

var (
	SecretName          string
	Ca                  *pem.Block
	ServiceAccountToken string
	ArgoPass            string
	ArgoToken           string
	VaultToken          string
)

type ClusterInstance struct {
	Name            string `json:"name"`
	ApiServer       string `json:"apiServer"`
	ClusterType     string `json:"clusterType"`
	HostCluster     string `json:"hostType"`
	Provider        string `json:"provider"`
	CertificateData string `json:"certficateData"`
}

//+kubebuilder:rbac:groups=resource.nautes.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=resource.nautes.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=resource.nautes.io,resources=clusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Cluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = context.Background()
	log := log.FromContext(ctx)

	log.Info("Start Reconcile !")
	cluster := &argov1alpha1.Cluster{}
	err := r.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		log.Error(err, "The cluster is not found, err: \t")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// add finalizer
	FinalizerName := "storage.finalizers.resource.nautes.io"
	if cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if !utilstring.ContainsString(cluster.ObjectMeta.Finalizers, FinalizerName) {
			log.Info("The cluster need append finalizer")
			cluster.ObjectMeta.Finalizers = append(cluster.ObjectMeta.Finalizers, FinalizerName)
			if err := r.Update(context.Background(), cluster); err != nil {
				log.Info("The cluster has been added finalizer")
				return ctrl.Result{}, err
			}
		}
	} else {
		log.Info("The cluster is being deleted")
		if utilstring.ContainsString(cluster.ObjectMeta.Finalizers, FinalizerName) {
			password, err := r.getArgoPassword()
			if err != nil {
				log.Error(err, "unable to get password form argocd")
				return ctrl.Result{}, err
			}
			token, err := getArgocdToken(password)
			if err != nil {
				log.Error(err, "unable to get token form argocd")
				return ctrl.Result{}, err
			}
			resBody, err := r.deleteArgocdCluster(token, password, cluster)
			if err != nil {
				return ctrl.Result{}, err
			}
			log.Info("delete argocd cluster result: " + resBody)

			cluster.ObjectMeta.Finalizers = utilstring.RemoveString(cluster.ObjectMeta.Finalizers, FinalizerName)
			if err := r.Update(context.Background(), cluster); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, err
		}
	}

	if !cluster.Status.Sync {
		is := reflect.ValueOf(cluster.Spec)
		clusterInstance := &ClusterInstance{
			Name:        is.FieldByName("Name").String(),
			ApiServer:   is.FieldByName("ApiServer").String(),
			ClusterType: is.FieldByName("ClusterType").String(),
			HostCluster: is.FieldByName("HostCluster").String(),
			Provider:    is.FieldByName("Provider").String(),
		}
		vt, err := r.getVaultToken()
		if err != nil {
			return ctrl.Result{}, err
		}
		crt, err := WithVaultSecret(vt, clusterInstance.Name)
		if err != nil {
			log.Error(err, "unable to get secret from valut")
			return ctrl.Result{}, err
		}
		clusterInstance.CertificateData = crt

		password, err := r.getArgoPassword()
		if err != nil {
			log.Error(err, "unable to get password from argocd")
			return ctrl.Result{}, err
		}
		token, err := getArgocdToken(password)
		if err != nil {
			log.Error(err, "unable to get token from argocd")
			return ctrl.Result{}, nil
		}
		resBody, err := r.syncArgocdCluster(token, clusterInstance, cluster)
		if err != nil {
			log.Error(err, "Failed to Sync cluster info")
			return ctrl.Result{}, err
		}
		log.Info("Sync cluster result is: \t" + resBody)
	} else {
		log.Info("Cluster synchronized")
	}

	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) getVaultToken() (string, error) {
	saList := &corev1.ServiceAccountList{}
	err := r.List(context.Background(), saList, &client.ListOptions{
		Namespace: ARGOCDNAMESPACE,
	})
	if err != nil {
		return "", err
	}
	for _, v := range saList.Items {
		name := v.ObjectMeta.Name
		ns := v.ObjectMeta.Namespace
		if name == "default" && ns == ARGOCDNAMESPACE {
			SecretName = v.Secrets[0].Name
			break
		}
	}

	secretList := &corev1.SecretList{}
	err = r.List(context.Background(), secretList, &client.ListOptions{
		Namespace: ARGOCDNAMESPACE,
	})
	if err != nil {
		return "", err
	}

	for _, v := range secretList.Items {
		name := v.ObjectMeta.Name
		ns := v.ObjectMeta.Namespace
		if name == SecretName && ns == ARGOCDNAMESPACE {
			t := v.Data["token"]
			return string(t), nil
		}
	}
	return "", nil
}

func (r *ClusterReconciler) deleteArgocdCluster(token string, pass string, cluster *argov1alpha1.Cluster) (string, error) {
	var argocdaddress = os.Getenv("ARGOCDADDRESS")
	if argocdaddress == "" {
		return "", fmt.Errorf("argocd address address not found")
	}
	clusterName := url.QueryEscape(cluster.Spec.ApiServer)
	url := argocdaddress + "/api/v1/clusters/" + clusterName

	cookies := &http.Cookie{
		Name:  "argocd.token",
		Value: token,
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	var req *http.Request
	req, _ = http.NewRequest("DELETE", url, nil)
	req.AddCookie(cookies)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)
	return string(b), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&resourcev1alpha1.Cluster{}).
		Complete(r)
}

// all logic here
var (
	vaultHost                string
	vaultCAPath              string
	vaultServiceAccount      string
	vaultJWTPath             string
	certificateAuthorityData string
	argoToken                string
)

// Get crt Using vault secret
func WithVaultSecret(jwt string, cluster string) (string, error) {
	vaultJWTPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	vaultServiceAccount = "argo-operator-cluster"

	config := vault.DefaultConfig() // modify for more granular configuration
	config.Address = "https://10.204.118.215:8200"

	client, err := vault.NewClient(config)

	if err != nil {
		return "", fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	k8sAuth, err := auth.NewKubernetesAuth(
		vaultServiceAccount,
		auth.WithServiceAccountToken(jwt),
		auth.WithMountPath(cluster),
	)
	if err != nil {
		return "", fmt.Errorf("unable to initialize Kubernetes auth method: %w", err)
	}

	authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		return "", fmt.Errorf("unable to log in with Kubernetes auth: %w", err)
	}
	if authInfo == nil {
		return "", fmt.Errorf("no auth info was returned after login")
	}

	// get secret from Vault, from the default mount path for KV v2 in dev mode, "secret"
	secret, err := client.KVv2("cluster").Get(context.Background(), "kubernetes/"+cluster+"/argo-operator/admin")
	if err != nil {
		return "", fmt.Errorf("unable to read secret: %w", err)
	}
	certificateAuthorityData = secret.Data["certificate-authority-data"].(string)

	return certificateAuthorityData, nil
}

type argoUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type argoResp struct {
	Token string `json:"token"`
}

func (r *ClusterReconciler) getArgoPassword() (string, error) {
	secretList := &corev1.SecretList{}
	err := r.List(context.Background(), secretList, &client.ListOptions{
		Namespace: ARGOCDNAMESPACE,
	})
	if err != nil {
		return "", err
	}

	for _, v := range secretList.Items {
		name := v.ObjectMeta.Name
		ns := v.ObjectMeta.Namespace
		if name == ARGOCDINITIALADMINSECRET && ns == ARGOCDNAMESPACE {
			p := v.Data["password"]
			ArgoPass = string(p)
			break
		}
	}
	return ArgoPass, nil
}

func getArgocdToken(pass string) (string, error) {
	if pass == "" {
		return "", fmt.Errorf("argocd pass not found")
	}
	var argocdaddress = os.Getenv("ARGOCDADDRESS")
	var req *http.Request
	user := &argoUser{
		Username: "admin",
		Password: pass,
	}
	userJSON, _ := json.Marshal(user)
	url := argocdaddress + "/api/v1/session"
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	req, _ = http.NewRequest("POST", url, bytes.NewReader(userJSON))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ap argoResp
	json.Unmarshal(body, &ap)
	argoToken = ap.Token
	return argoToken, nil
}

type tlsClientConfig struct {
	Insecure bool   `json:"insecure"`
	CaData   string `json:"caData"`
}

type clusterInfoConfig struct {
	TlsClientConfig tlsClientConfig `json:"tlsClientConfig"`
}

type Info struct {
	serverVersion string
}

type ClusterInfo struct {
	Server string            `json:"server"`
	Name   string            `json:"name"`
	Config clusterInfoConfig `json:"config"`
	Info   Info              `json:"info"`
}

func (r *ClusterReconciler) syncArgocdCluster(token string, clusterInstance *ClusterInstance, cluster *argov1alpha1.Cluster) (string, error) {
	// data cluster
	data := &ClusterInfo{
		Server: clusterInstance.ApiServer,
		Name:   clusterInstance.Name,
		Config: clusterInfoConfig{
			TlsClientConfig: tlsClientConfig{
				Insecure: true,
				// CaData:   clusterInstance.CertificateData,
			},
		},
		Info: Info{
			serverVersion: "1.21",
		},
	}

	datajson, _ := json.Marshal(data)
	//argocd api
	var argocdaddress = os.Getenv("ARGOCDADDRESS")
	if argocdaddress == "" {
		return "", fmt.Errorf("argocd address is not found")
	}
	url := argocdaddress + "/api/v1/clusters?upsert=true"
	cookies := &http.Cookie{
		Name:  "argocd.token",
		Value: token,
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	var req *http.Request
	req, _ = http.NewRequest("POST", url, bytes.NewReader(datajson))
	req.AddCookie(cookies)
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := ioutil.ReadAll(resp.Body)

	// update resource
	cluster.Status.Sync = true
	if err := r.Status().Update(context.Background(), cluster); err != nil {
		return "", err
	}
	return string(b), nil
}

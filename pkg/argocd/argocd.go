package argocd

import (
	"argo-operator/pkg/client"
	"argo-operator/pkg/request"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

const (
	NAMESPACE    = "argocd"
	USERNAME     = "admin"
	ADMINSECRET  = "argocd-initial-admin-secret"
	ARGOCDADRESS = "https://10.204.118.217:32531/"
)

func GetPassword() (string, error) {
	result := &corev1.SecretList{}
	namespace := NAMESPACE
	restClient, err := client.NewRestClient()
	if err != nil {
		panic(err)
	}
	err = restClient.Get().
		Namespace(namespace).
		Resource("secrets").
		VersionedParams(&metav1.ListOptions{Limit: 100}, scheme.ParameterCodec).
		Do().
		Into(result)

	if err != nil {
		panic(err.Error())
	}
	fmt.Println("test")
	password := ""
	for _, d := range result.Items {
		if d.Name == ADMINSECRET {
			p, ok := d.Data["password"]
			if !ok {
				panic(ok)
			}
			password = string(p)
		}
	}
	return password, nil
}

func GetToken() (string, error) {
	password, err := GetPassword()
	if err != nil {
		fmt.Println(err.Error())
		panic(err)
	}
	data := make(map[string]interface{})
	data["username"] = USERNAME
	data["password"] = password
	url := ARGOCDADRESS + "api/v1/session"
	req := request.NewRequest(url)
	body, err := req.PostHttps(data)
	if err != nil {
		panic(err)
	}
	type Users struct {
		Token string `json:"token"`
	}
	var users Users
	json.Unmarshal(body, &users)
	fmt.Printf("token is %v\t %v\n", string(body), users.Token)
	return "", nil
}

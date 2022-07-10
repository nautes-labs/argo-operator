package client

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
)

func TestNewClient(t *testing.T) {
	NewRestClient()
}

func TestGetService(t *testing.T) {
	restClient, err := NewRestClient()
	if err != nil {
		panic(err)
	}
	namespace := "argocd"
	result := &corev1.ServiceList{}
	err = restClient.Get().
		//  指定namespace，参考path : /api/v1/namespaces/{namespace}/pods
		Namespace(namespace).
		// 查找多个pod，参考path : /api/v1/namespaces/{namespace}/pods
		Resource("services").
		// 指定大小限制和序列化工具
		VersionedParams(&metav1.ListOptions{Limit: 100}, scheme.ParameterCodec).
		// 请求
		Do().
		// 结果存入result
		Into(result)
	if err != nil {
		panic(err)
	}
	// fmt.Printf("%v \n", result.Kind)
	for _, d := range result.Items {
		if d.Name == "argocd-server" {
			nodeport := d.Spec.Ports[0].NodePort
			fmt.Printf("%v", nodeport)
		}
	}
}

func TestGetIngress(t *testing.T) {
	clientset, _ := NewClient()
	ingressResult, _ := clientset.ExtensionsV1beta1().Ingresses("platform-management").Get("vcluster-platform", metav1.GetOptions{})
	fmt.Printf("%s", ingressResult.Name)
}

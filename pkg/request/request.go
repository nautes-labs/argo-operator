package request

import (
	"argo-operator/pkg/util"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Request struct {
	url string
}

func NewRequest(url string) *Request {
	r := &Request{
		url: url,
	}
	return r
}

func NewHttpClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	return client
}

func (r *Request) GetHttps() (string, error) {
	url := r.url
	client := NewHttpClient()
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer client.CloseIdleConnections()
	sb := util.Bytes2String(body)
	return sb, nil
}

func (r *Request) PostHttps(data map[string]interface{}) ([]byte, error) {
	url := r.url
	contentType := "application/json"
	bytesData, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err.Error())
	}

	client := NewHttpClient()
	resp, err := client.Post(url, contentType, bytes.NewReader(bytesData))
	if err != nil {
		fmt.Println(err.Error())
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, nil
}

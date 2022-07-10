package request

import (
	"fmt"
	"testing"
)

func TestGetHttps(t *testing.T) {
	r := Request{
		url: "https://10.204.118.216:31680",
	}
	body, _ := r.GetHttps()
	fmt.Printf("%s", body)
}

func TestPostHttps(t *testing.T) {
	data := make(map[string]interface{})
	data["username"] = "admin"
	data["password"] = "wlLuw0kRM71UQoQV"
	r := &Request{
		url: "https://10.204.118.216:32310/api/v1/session",
	}
	body, _ := r.PostHttps(data)
	fmt.Printf("%v", body)
}

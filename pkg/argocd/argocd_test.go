package argocd

import (
	"fmt"
	"testing"
)

func TestGetArgocdPaswword(t *testing.T) {
	password, _ := GetPassword()
	fmt.Printf("%s\n", password)
}

func TestGetToken(t *testing.T) {
	body, _ := GetToken()
	fmt.Printf("%s\n", body)
}

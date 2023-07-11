package secret

import (
	nautesconfigs "github.com/nautes-labs/pkg/pkg/nautesconfigs"
)

type SecretConfig struct {
	SecretRepo *nautesconfigs.SecretRepo
	Namespace  string
}

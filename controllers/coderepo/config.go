package coderepo

import (
	nautesconfigs "github.com/nautes-labs/pkg/pkg/nautesconfigs"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetNautesConfigs(c client.Client) (nautesConfigs *nautesconfigs.Config, err error) {
	config := nautesconfigs.NautesConfigs{}
	nautesConfigs, err = config.GetConfigByClient(c)
	if err != nil {
		return
	}
	return
}

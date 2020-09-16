package configuration

import (
	"errors"
	"github.com/shima-park/agollo"
	"os"
)

type ConfigurationLoader interface {
	GetField(svcName, configKey string) string
	GetFieldError(svcName, configKey string) (string, error)
}

type ApolloConfigurationProvider struct {
	configServer string
	cluster      string
	appId        string
	namespace    string
	ip           string
	client       agollo.Agollo
}

func (acp *ApolloConfigurationProvider) Initialize() {
	acp.configServer = os.Getenv("CONFIG_SERVER")
	acp.appId = os.Getenv("APP_ID")
	if acp.appId == "" {
		acp.appId = "moebius"
	}
	acp.cluster = os.Getenv("ENV")
	if acp.cluster == "" {
		acp.cluster = "test"
	}
	acp.namespace = os.Getenv("NAMESPACE")
	if acp.namespace == "" {
		acp.namespace = "DevOps.Mgmt.API"
	}
	acp.ip = os.Getenv("SVC_NAME")

	var c agollo.Option
	if acp.ip != "" {
		c = agollo.WithApolloClient(agollo.NewApolloClient(agollo.WithIP(acp.ip)))
	} else {
		c = agollo.WithApolloClient(agollo.NewApolloClient())
	}

	client, err := agollo.New(acp.configServer, acp.appId, c, agollo.FailTolerantOnBackupExists(), agollo.AutoFetchOnCacheMiss(), agollo.Cluster(acp.cluster), agollo.DefaultNamespace(acp.namespace))
	if err != nil {
		panic(err)
	}
	acp.client = client
	client.Start()
}

func (acp *ApolloConfigurationProvider) GetField(svcName, configKey string) string {
	value := acp.client.Get(configKey, agollo.WithNamespace(svcName))
	return value
}

func (acp *ApolloConfigurationProvider) GetFieldError(svcName, configKey string) (string, error) {
	value := acp.client.Get(configKey, agollo.WithNamespace(svcName))
	if value == "" {
		return "", errors.New("not found")
	}
	return value, nil
}

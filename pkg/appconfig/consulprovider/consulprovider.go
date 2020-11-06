package consulprovider

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/appconfig"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/consul"
)

const (
	ConfigRootConsulConfigProvider = "spring.cloud.consul.config"
	ConfigKeyAppName               = "spring.application.name"
)

type ConsulConfigProperties struct {
	Enabled        bool   `json:"enabled"`
	Prefix         string `json:"prefix"`
	DefaultContext string `json:"default-context"`
}

type ConfigProvider struct {
	appconfig.ProviderMeta
	contextPath  string
	connection   *consul.Connection
}

func (configProvider *ConfigProvider) Load() (loadError error) {
	defer func(){
		if loadError != nil {
			configProvider.IsLoaded = false
		} else {
			configProvider.IsLoaded = true
		}
	}()

	configProvider.Settings = make(map[string]interface{})

	// load keys from default context
	var defaultSettings map[string]interface{}

	defaultSettings, loadError = configProvider.connection.ListKeyValuePairs(configProvider.contextPath)
	if loadError != nil {
		return loadError
	}

	unFlattenedSettings, loadError := appconfig.UnFlatten(defaultSettings)
	if loadError != nil {
		return loadError
	}

	configProvider.Settings = unFlattenedSettings

	return nil
}

func NewConsulProvider(description string, precedence int, contextPath string, conn *consul.Connection) *ConfigProvider {
	return &ConfigProvider{
			ProviderMeta: appconfig.ProviderMeta{Description: description, Precedence: precedence},
			contextPath:  contextPath, //fmt.Sprintf("%s/%s", f.sourceConfig.Prefix, f.contextPath)
			connection:   conn,
		}
}
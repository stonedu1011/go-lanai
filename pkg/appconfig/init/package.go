package init

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/appconfig"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/appconfig/commandprovider"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/appconfig/consulprovider"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/appconfig/fileprovider"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/bootstrap"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/consul"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

var ConfigModule = &bootstrap.Module{
	Precedence: bootstrap.HighestPrecedence,
	PriorityOptions: []fx.Option{
		fx.Provide(
			newCommandProvider,
			newBootstrapFileProvider,
			newBootstrapConfig,
			newApplicationFileProvider,
			newConsulProvider,
			newConsulConfigProperties,
			newApplicationConfig),
	},
}

func init() {
	bootstrap.Register(ConfigModule)
}

const (
	//TODO: need to leave space for app specific and profile specific providers
	ConsulAppPrecedence            = iota
	ConsulDefaultPrecedence        = iota
	CommandlinePrecedence          = iota
	ApplicationLocalFilePrecedence = iota
	BootstrapLocalFilePrecedence   = iota
)

func newCommandProvider(cmd *cobra.Command) *commandprovider.ConfigProvider {
	p := commandprovider.NewCobraProvider("command line", CommandlinePrecedence, cmd, "cli.flag.")
	return p
}

type bootstrapFileProviderResult struct {
	fx.Out
	FileProvider *fileprovider.ConfigProvider `name:"bootstrap_file_provider"`
}

func newBootstrapFileProvider() bootstrapFileProviderResult {
	//TODO: one each for app specific and profile specific providers
	p := fileprovider.NewFileProvidersFromBaseName("bootstrap file properties", BootstrapLocalFilePrecedence, "bootstrap", "yml")
	return bootstrapFileProviderResult{FileProvider: p}
}

type bootstrapConfigParam struct {
	fx.In
	CmdProvider  *commandprovider.ConfigProvider
	FileProvider *fileprovider.ConfigProvider `name:"bootstrap_file_provider"`
}

func newBootstrapConfig(p bootstrapConfigParam) *appconfig.BootstrapConfig {
	bootstrapConfig := appconfig.NewBootstrapConfig(p.FileProvider, p.CmdProvider)

	error := bootstrapConfig.Load(false)
	if error != nil {
		panic(error)
	}

	return bootstrapConfig
}

func newConsulConfigProperties(bootstrapConfig *appconfig.BootstrapConfig) *consulprovider.ConsulConfigProperties {
	p := &consulprovider.ConsulConfigProperties{
		Prefix: "userviceconfiguration",
		DefaultContext: "defaultapplication",
	}
	bootstrapConfig.Bind(p, consulprovider.ConfigRootConsulConfigProvider)
	return p
}

type consulProviderResults struct {
	fx.Out
	Providers []appconfig.Provider `name:consul_providers`
}

func newConsulProvider(	bootstrapConfig *appconfig.BootstrapConfig, consulConfigProperties *consulprovider.ConsulConfigProperties, consulConnection *consul.Connection) consulProviderResults {
	appName := bootstrapConfig.Value(consulprovider.ConfigKeyAppName)

	//TODO: profile specific ones

	//1. default contexts
	defaultContextConsulProvider := consulprovider.NewConsulProvider(
		"consul provider - default context",
		ConsulDefaultPrecedence,
		fmt.Sprintf("%s/%s", consulConfigProperties.Prefix, consulConfigProperties.DefaultContext),
		consulConnection,
	)
	applicationContextConsulProvider := consulprovider.NewConsulProvider(
		"consul provider - app specific context",
		ConsulAppPrecedence,
		fmt.Sprintf("%s/%s", consulConfigProperties.Prefix, appName),
		consulConnection,
	)
	return consulProviderResults{Providers: []appconfig.Provider{defaultContextConsulProvider, applicationContextConsulProvider}}
}

type applicationFileProviderResult struct {
	fx.Out
	FileProvider *fileprovider.ConfigProvider `name:"application_file_provider"`
}

func newApplicationFileProvider() applicationFileProviderResult {
	//TODO: return a list of these for application specific, and one for each profile
	p := fileprovider.NewFileProvidersFromBaseName("application file properties", ApplicationLocalFilePrecedence, "application", "yml")
	return applicationFileProviderResult{FileProvider: p}
}

type newApplicationConfigParam struct {
	fx.In
	FileProvider       *fileprovider.ConfigProvider `name:"application_file_provider"`
	ConsulProviders	   []appconfig.Provider      `name:consul_providers`
	BootstrapConfig    *appconfig.BootstrapConfig
}

func newApplicationConfig(p newApplicationConfigParam) *appconfig.ApplicationConfig {
	var mergedProvider []appconfig.Provider

	mergedProvider = append(mergedProvider, p.FileProvider)
	mergedProvider = append(mergedProvider, p.ConsulProviders...)
	mergedProvider = append(mergedProvider, p.BootstrapConfig.Providers...)

	applicationConfig := appconfig.NewApplicationConfig(mergedProvider...)

	error := applicationConfig.Load(false)

	if error != nil {
		panic(error)
	}

	return applicationConfig
}

// Maker func, does nothing. Allow service to include this module in main()
func Use() {

}
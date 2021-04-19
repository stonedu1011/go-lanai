package appconfig

import (
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/appconfig"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/appconfig/cliprovider"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/appconfig/envprovider"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/appconfig/fileprovider"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/bootstrap"
	"fmt"
	"go.uber.org/fx"
)

type bootstrapProvidersOut struct {
	fx.Out
	ProviderGroup appconfig.ProviderGroup `group:"bootstrap-config"`
}

type appConfigProvidersOut struct {
	fx.Out
	ProviderGroup appconfig.ProviderGroup `group:"application-config"`
}

/*********************
	Bootstrap Groups
 *********************/

func newCommandProviderGroup(execCtx *bootstrap.CliExecContext) bootstrapProvidersOut {
	p := cliprovider.NewCobraProvider(commandlinePrecedence, execCtx, "cli.flag.")
	return bootstrapProvidersOut {
		ProviderGroup: appconfig.NewStaticProviderGroup(commandlinePrecedence, p),
	}
}

func newOsEnvProviderGroup() bootstrapProvidersOut {
	p := envprovider.NewEnvProvider(osEnvPrecedence)
	return bootstrapProvidersOut {
		ProviderGroup: appconfig.NewStaticProviderGroup(osEnvPrecedence, p),
	}
}

func newBootstrapFileProviderGroup() bootstrapProvidersOut {
	const name = "bootstrap"
	const ext = "yml"
	group := appconfig.NewProfileBasedProviderGroup(bootstrapLocalFilePrecedence)
	group.KeyFunc = func(profile string) string {
		if profile == "" {
			return name
		}
		return fmt.Sprintf("%s-%s", name, profile)
	}
	group.CreateFunc = func(name string, order int, conf bootstrap.ApplicationConfig) appconfig.Provider {
		ptr, exists := fileprovider.NewFileProvidersFromBaseName(order, name, ext, conf)
		if !exists || ptr == nil {
			return nil
		}
		return ptr
	}
	group.ProcessFunc = func(ctx context.Context, providers []appconfig.Provider) []appconfig.Provider {
		if len(providers) == 0 {
			logger.Warnf("no bootstrap configuration file found. are you running from the project root directory?")
		}
		return providers
	}

	return bootstrapProvidersOut {
		ProviderGroup: group,
	}
}

func newDefaultProviderGroup(execCtx *bootstrap.CliExecContext) bootstrapProvidersOut {
	p := cliprovider.NewStaticConfigProvider(defaultPrecedence, execCtx)
	return bootstrapProvidersOut {
		ProviderGroup: appconfig.NewStaticProviderGroup(defaultPrecedence, p),
	}
}

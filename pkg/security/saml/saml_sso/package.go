package saml_auth

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/bootstrap"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/log"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/saml"
	saml_auth_ctx "cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/saml/saml_sso/saml_sso_ctx"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web"
	"go.uber.org/fx"
)

var Module = &bootstrap.Module{
	Name: "saml auth - authorize",
	Precedence: security.MinSecurityPrecedence + 20,
	Options: []fx.Option{
		fx.Invoke(register),
	},
}

var logger = log.New("SAML.SSO")

func Use() {
	bootstrap.Register(Module)
}

type initDI struct {
	fx.In
	SecRegistrar           security.Registrar `optional:"true"`
	Properties             saml.SamlProperties
	ServerProperties       web.ServerProperties
	ServiceProviderManager saml_auth_ctx.SamlClientStore `optional:"true"`
	AccountStore           security.AccountStore `optional:"true"`
	AttributeGenerator     AttributeGenerator `optional:"true"`
}

func register(di initDI) {
	if di.SecRegistrar != nil {
		authConfigurer := newSamlAuthorizeEndpointConfigurer(di.Properties,
			di.ServiceProviderManager, di.AccountStore,
			di.AttributeGenerator)
		di.SecRegistrar.(security.FeatureRegistrar).RegisterFeature(FeatureId, authConfigurer)

		sloConfigurer := newSamlLogoutEndpointConfigurer(di.Properties, di.ServiceProviderManager)
		di.SecRegistrar.(security.FeatureRegistrar).RegisterFeature(SloFeatureId, sloConfigurer)
	}
}
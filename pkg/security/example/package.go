package example

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/bootstrap"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/discovery"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/log"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/config/authserver"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/config/resserver"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/idp"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/idp/passwdidp"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/idp/samlidp"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/idp/unknownIdp"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/oauth2"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/passwd"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web/matcher"
	"go.uber.org/fx"
	"net/url"
)

var logger = log.New("SEC.Example")

func init() {

}

// Maker func, does nothing. Allow service to include this module in main()
func Use() {
	authserver.Use()
	resserver.Use()
	bootstrap.AddOptions(
		fx.Provide(BindAccountsProperties),
		fx.Provide(BindAccountPoliciesProperties),
		fx.Provide(BindClientsProperties),
		fx.Provide(NewInMemoryAccountStore),
		fx.Provide(NewInMemoryFederatedAccountStore),
		fx.Provide(NewInMemoryClientStore),
		fx.Provide(NewTenantStore),
		fx.Provide(NewProviderStore),
		fx.Provide(NewInMemoryIdpManager),
		fx.Provide(NewInMemSpManager),
		fx.Provide(newAuthServerConfigurer),
		fx.Invoke(configureSecurity),
		fx.Invoke(configureConsulRegistration),
	)
}

func configureSecurity(init security.Registrar) {
	init.Register(&ErrorPageSecurityConfigurer{})
}

type authDI struct {
	fx.In
	ClientStore   oauth2.OAuth2ClientStore
	AccountStore  security.AccountStore
	TenantStore   security.TenantStore
	ProviderStore security.ProviderStore
	IdpManager    idp.IdentityProviderManager
	// TODO properties
}

func newAuthServerConfigurer(di authDI) authserver.AuthorizationServerConfigurer {
	return func(config *authserver.Configuration) {
		config.AddIdp(passwdidp.NewPasswordIdpSecurityConfigurer())
		config.AddIdp(samlidp.NewSamlIdpSecurityConfigurer())
		config.AddIdp(unknownIdp.NewNoIdpSecurityConfigurer())

		config.IdpManager = di.IdpManager
		config.ClientStore = di.ClientStore
		config.ClientSecretEncoder = passwd.NewNoopPasswordEncoder()
		config.UserAccountStore = di.AccountStore
		config.TenantStore = di.TenantStore
		config.ProviderStore = di.ProviderStore
		config.UserPasswordEncoder = passwd.NewNoopPasswordEncoder()
		config.Endpoints = authserver.Endpoints{
			Authorize: authserver.ConditionalEndpoint{
				Location: &url.URL{Path: "/v2/authorize"},
				Condition: matcher.NotRequest(matcher.RequestWithParam("grant_type", "urn:ietf:params:oauth:grant-type:saml2-bearer")),
			},
			Approval: "/v2/approve",
			Token: "/v2/token",
			CheckToken: "/v2/check_token",
			UserInfo: "/v2/userinfo",
			JwkSet: "/v2/jwks",
			Logout: "/v2/logout",
			SamlSso: authserver.ConditionalEndpoint{
				Location: &url.URL{Path:"/v2/authorize", RawQuery: "grant_type=urn:ietf:params:oauth:grant-type:saml2-bearer"},
				Condition: matcher.RequestWithParam("grant_type", "urn:ietf:params:oauth:grant-type:saml2-bearer"),
			},
			SamlMetadata: "/metadata",
		}
	}
}

func configureConsulRegistration(r *discovery.Customizers) {
	r.Add(&RegistrationCustomizer{})
}
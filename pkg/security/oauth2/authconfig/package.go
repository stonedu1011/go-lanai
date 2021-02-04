package authconfig

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/bootstrap"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"go.uber.org/fx"
)

//goland:noinspection GoNameStartsWithPackageName
var OAuth2AuthorizeModule = &bootstrap.Module{
	Name: "oauth2 authconfig",
	Precedence: security.MinSecurityPrecedence + 20,
	Options: []fx.Option{
		fx.Invoke(ConfigureAuthorizationServer),
	},
}

func init() {
	bootstrap.Register(OAuth2AuthorizeModule)
}

func Use() {

}

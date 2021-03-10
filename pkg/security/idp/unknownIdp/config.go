package unknownIdp

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/access"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/config/authserver"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/csrf"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/errorhandling"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/idp"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/logout"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/redirect"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/session"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/web/matcher"
)

type NoIdpSecurityConfigurer struct {
}

func NewNoIdpSecurityConfigurer() *NoIdpSecurityConfigurer {
	return &NoIdpSecurityConfigurer{}
}

func (c *NoIdpSecurityConfigurer) Configure(ws security.WebSecurity, config *authserver.Configuration) {
	// For Authorize endpoint
	handler := redirect.NewRedirectWithRelativePath("/error")
	condition := idp.RequestWithAuthenticationFlow(idp.UnknownIdp, config.IdpManager)

	ws.AndCondition(condition).
		With(session.New()).
		With(access.New().
			Request(matcher.AnyRequest()).Authenticated(),
		).
		With(logout.New().
			LogoutUrl(config.Endpoints.Logout),
			// TODO SSO logout success handler
			//SuccessHandler()
		).
		With(errorhandling.New().
			AccessDeniedHandler(handler),
		).
		With(csrf.New().IgnoreCsrfProtectionMatcher(matcher.RequestWithPattern(config.Endpoints.Authorize.Location.Path)),
	)
}
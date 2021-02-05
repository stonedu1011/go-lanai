package authserver

import (
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/redis"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/oauth2"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/oauth2/auth"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/oauth2/auth/grants"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/oauth2/common"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/oauth2/jwt"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/passwd"
)

type AuthorizationServerConfigurer func(*Configuration)

// Configuration entry point
func ConfigureAuthorizationServer(registrar security.Registrar, configurer AuthorizationServerConfigurer) {
	config := &Configuration{}
	configurer(config)

	registrar.Register(&ClientAuthEndpointsConfigurer{config: config})
	for _, configuer := range config.idpConfigurers {
		registrar.Register(&AuthorizeEndpointConfigurer{config: config, delegate: configuer})
	}
}

/****************************
	configuration
 ****************************/
type Endpoints struct {
	Authorize  string
	Token      string
	CheckToken string
	UserInfo   string
	JwkSet     string
	Logout     string
}

type Configuration struct {
	// configurable items
	ClientStore         oauth2.OAuth2ClientStore
	ClientSecretEncoder passwd.PasswordEncoder
	Endpoints           Endpoints
	UserAccountStore    security.AccountStore
	TenantStore         security.TenantStore
	ProviderStore       security.ProviderStore
	UserPasswordEncoder passwd.PasswordEncoder
	TokenStore          auth.TokenStore
	JwkStore            jwt.JwkStore
	RedisClientFactory  redis.ClientFactory

	// not directly configurable items
	idpConfigurers              []IdpSecurityConfigurer
	sharedErrorHandler          *auth.OAuth2ErrorHandler
	sharedTokenGranter          auth.TokenGranter
	sharedAuthService           auth.AuthorizationService
	sharedPasswordAuthenticator security.Authenticator
	sharedContextDetailsStore   security.ContextDetailsStore
	sharedJwtEncoder            jwt.JwtEncoder
	sharedJwtDecoder            jwt.JwtDecoder
	sharedDetailsFactory        *common.ContextDetailsFactory
	sharedARProcessor           auth.AuthorizeRequestProcessor

	// TODO
}

func (c *Configuration) AddIdp(configurer IdpSecurityConfigurer) {
	c.idpConfigurers = append(c.idpConfigurers, configurer)
}

func (c *Configuration) clientSecretEncoder() passwd.PasswordEncoder {
	if c.ClientSecretEncoder == nil {
		c.ClientSecretEncoder = passwd.NewNoopPasswordEncoder()
	}
	return c.ClientSecretEncoder
}

func (c *Configuration) userPasswordEncoder() passwd.PasswordEncoder {
	if c.UserPasswordEncoder == nil {
		c.UserPasswordEncoder = passwd.NewNoopPasswordEncoder()
	}
	return c.UserPasswordEncoder
}

func (c *Configuration) errorHandler() *auth.OAuth2ErrorHandler {
	if c.sharedErrorHandler == nil {
		c.sharedErrorHandler = auth.NewOAuth2ErrorHanlder()
	}
	return c.sharedErrorHandler
}

func (c *Configuration) tokenGranter() auth.TokenGranter {
	if c.sharedTokenGranter == nil {
		granters := []auth.TokenGranter {
			grants.NewAuthorizationCodeGranter(c.authorizationService()),
			grants.NewClientCredentialsGranter(c.authorizationService()),
		}

		// password granter is optional
		if c.passwordGrantAuthenticator() != nil {
			passwordGranter := grants.NewPasswordGranter(c.passwordGrantAuthenticator(), c.authorizationService())
			granters = append(granters, passwordGranter)
		}

		c.sharedTokenGranter = auth.NewCompositeTokenGranter(granters...)
	}
	return c.sharedTokenGranter
}

func (c *Configuration) passwordGrantAuthenticator() security.Authenticator {
	if c.sharedPasswordAuthenticator == nil && c.UserAccountStore != nil {
		authenticator, err := passwd.NewAuthenticatorBuilder(
			passwd.New().
				AccountStore(c.UserAccountStore).
				PasswordEncoder(c.userPasswordEncoder()).
				MFA(false),
		).Build(context.Background())

		if err == nil {
			c.sharedPasswordAuthenticator = authenticator
		}
	}
	return c.sharedPasswordAuthenticator
}

func (c *Configuration) contextDetailsStore() security.ContextDetailsStore {
	if c.sharedContextDetailsStore == nil {
		c.sharedContextDetailsStore = common.NewRedisContextDetailsStore(c.RedisClientFactory)
	}
	return c.sharedContextDetailsStore
}

func (c *Configuration) tokenStore() auth.TokenStore {
	if c.TokenStore == nil {
		c.TokenStore = auth.NewJwtTokenStore(func(opt *auth.JTSOption) {
			opt.DetailsStore = c.contextDetailsStore()
			opt.Encoder = c.jwtEncoder()
			opt.Decoder = c.jwtDecoder()
			// TODO enhancers
		})
	}
	return c.TokenStore
}

func (c *Configuration) authorizationService() auth.AuthorizationService {
	if c.sharedAuthService == nil {
		c.sharedAuthService = auth.NewDefaultAuthorizationService(func(conf *auth.DASOption) {
			conf.TokenStore = c.tokenStore()
			conf.DetailsFactory = c.contextDetailsFactory()
			conf.ClientStore = c.ClientStore
			conf.AccountStore = c.UserAccountStore
			conf.TenantStore = c.TenantStore
			conf.ProviderStore = c.ProviderStore
		})
	}

	return c.sharedAuthService
}

func (c *Configuration) jwkStore() jwt.JwkStore {
	if c.JwkStore == nil {
		// TODO
		c.JwkStore = jwt.NewStaticJwkStore("default")
	}
	return c.JwkStore
}

func (c *Configuration) jwtEncoder() jwt.JwtEncoder {
	if c.sharedJwtEncoder == nil {
		// TODO
		c.sharedJwtEncoder = jwt.NewRS256JwtEncoder(c.jwkStore(), "default")
	}
	return c.sharedJwtEncoder
}

func (c *Configuration) jwtDecoder() jwt.JwtDecoder {
	if c.sharedJwtDecoder == nil {
		// TODO
		c.sharedJwtDecoder = jwt.NewRS256JwtDecoder(c.jwkStore(), "default")
	}
	return c.sharedJwtDecoder
}

func (c *Configuration) contextDetailsFactory() *common.ContextDetailsFactory {
	if c.sharedDetailsFactory == nil {
		c.sharedDetailsFactory = common.NewContextDetailsFactory()
	}
	return c.sharedDetailsFactory
}

func (c *Configuration) authorizeRequestProcessor() auth.AuthorizeRequestProcessor {
	if c.sharedARProcessor == nil {
		//TODO OIDC overwrite
		std := auth.NewStandardAuthorizeRequestProcessor(func(opt *auth.StdARPOption) {
			opt.ClientStore = c.ClientStore
			opt.ResponseTypes = auth.StandardResponseTypes
		})
		c.sharedARProcessor = auth.NewCompositeAuthorizeRequestProcessor(std)
	}
	return c.sharedARProcessor
}


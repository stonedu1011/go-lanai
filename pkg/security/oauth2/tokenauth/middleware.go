package tokenauth

import (
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/oauth2"
	"errors"
	"github.com/gin-gonic/gin"
	"strings"
)

const (
	bearerTokenPrefix = "Bearer "
)

/****************************
	Token Authentication
 ****************************/

//goland:noinspection GoNameStartsWithPackageName
type TokenAuthMiddleware struct {
	authenticator   security.Authenticator
	successHandler  security.AuthenticationSuccessHandler
	postBodyEnabled bool
}

//goland:noinspection GoNameStartsWithPackageName
type TokenAuthMWOptions func(opt *TokenAuthMWOption)

//goland:noinspection GoNameStartsWithPackageName
type TokenAuthMWOption struct {
	Authenticator   security.Authenticator
	SuccessHandler  security.AuthenticationSuccessHandler
	PostBodyEnabled bool
}

func NewTokenAuthMiddleware(opts ...TokenAuthMWOptions) *TokenAuthMiddleware {
	opt := TokenAuthMWOption{}
	for _, optFunc := range opts {
		if optFunc != nil {
			optFunc(&opt)
		}
	}
	return &TokenAuthMiddleware{
		authenticator:   opt.Authenticator,
		successHandler:  opt.SuccessHandler,
		postBodyEnabled: opt.PostBodyEnabled,
	}
}

func (mw *TokenAuthMiddleware) AuthenticateHandlerFunc() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		// We always re-authenticate by clearing current auth
		before := security.Get(ctx)
		security.Clear(ctx)

		// grab bearer token and create candidate
		tokenValue, e := mw.extractAccessToken(ctx)
		if e != nil {
			mw.handleError(ctx, e)
			return
		} else if tokenValue == "" {
			// token is not present, we continue the MW chain
			return
		}
		candidate := BearerToken{
			Token:      tokenValue,
			DetailsMap: map[string]interface{}{},
		}

		// Authenticate
		auth, err := mw.authenticator.Authenticate(ctx, &candidate)
		if err != nil {
			mw.handleError(ctx, err)
			return
		}
		mw.handleSuccess(ctx, before, auth)
	}
}

func (mw *TokenAuthMiddleware) handleSuccess(c *gin.Context, before, new security.Authentication) {
	if new != nil {
		c.Set(gin.AuthUserKey, new.Principal())
		c.Set(security.ContextKeySecurity, new)
	}

	mw.successHandler.HandleAuthenticationSuccess(c, c.Request, c.Writer, before, new)
	// we don't explicitly write any thig on success
}

func (mw *TokenAuthMiddleware) extractAccessToken(ctx *gin.Context) (ret string, err error) {
	header := ctx.GetHeader("Authorization")
	if header == "" {
		if mw.postBodyEnabled {
			ret = ctx.PostForm(oauth2.ParameterAccessToken)
		}
		return
	}
	if !strings.HasPrefix(header, bearerTokenPrefix) {
		return "", oauth2.NewInvalidAccessTokenError("missing bearer token")
	}

	return strings.TrimPrefix(header, bearerTokenPrefix), nil
}

func (mw *TokenAuthMiddleware) handleError(c *gin.Context, err error) {
	if !errors.Is(err, oauth2.ErrorTypeOAuth2) {
		err = oauth2.NewInvalidAccessTokenError(err)
	}

	security.Clear(c)
	_ = c.Error(err)
	c.Abort()
}

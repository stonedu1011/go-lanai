package claims

import (
	"context"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/security/oauth2"
	"cto-github.cisco.com/NFV-BU/go-lanai/pkg/utils"
	"errors"
	"fmt"
)

var (
	errorInvalidSpec          = errors.New("invalid claim spec")
	errorMissingToken         = errors.New("source authentication is missing valid token")
	errorMissingRequest       = errors.New("source authentication is missing OAuth2 request")
	errorMissingUser          = errors.New("source authentication is missing user")
	errorMissingDetails       = errors.New("source authentication is missing required details")
	errorMissingClaims        = errors.New("source authentication is missing required token claims")
	errorMissingRequestParams = errors.New("source authentication's OAuth2 request is missing parameters")
)

type ClaimFactoryFunc func(ctx context.Context, opt *FactoryOption) (v interface{}, err error)
type ClaimRequirementFunc func(ctx context.Context, opt *FactoryOption) bool

type FactoryOptions func(opt *FactoryOption)

type FactoryOption struct {
	Specs           []map[string]ClaimSpec
	Source          oauth2.Authentication
	Issuer          security.Issuer
	AccountStore    security.AccountStore
	AccessToken     oauth2.AccessToken
	RequestedClaims RequestedClaims
	ClaimsFormula   []map[string]ClaimSpec
}

func WithSpecs(specs ...map[string]ClaimSpec) FactoryOptions {
	return func(opt *FactoryOption) {
		opt.Specs = append(opt.Specs, specs...)
	}
}

func WithRequestedClaims(requested RequestedClaims, formula ...map[string]ClaimSpec) FactoryOptions {
	return func(opt *FactoryOption) {
		opt.RequestedClaims = requested
		opt.ClaimsFormula = formula
	}
}

// WithSource is a FactoryOptions
func WithSource(oauth oauth2.Authentication) FactoryOptions {
	return func(opt *FactoryOption) {
		opt.Source = oauth
	}
}

func WithIssuer(issuer security.Issuer) FactoryOptions {
	return func(opt *FactoryOption) {
		opt.Issuer = issuer
	}
}

func WithAccountStore(accountStore security.AccountStore) FactoryOptions {
	return func(opt *FactoryOption) {
		opt.AccountStore = accountStore
	}
}

func WithAccessToken(token oauth2.AccessToken) FactoryOptions {
	return func(opt *FactoryOption) {
		opt.AccessToken = token
	}
}

func Populate(ctx context.Context, claims oauth2.Claims, opts ...FactoryOptions) error {
	opt := FactoryOption{}
	for _, fn := range opts {
		fn(&opt)
	}

	// populate based on specs
	for _, specs := range opt.Specs {
		if e := populateWithSpecs(ctx, claims, specs, &opt, nil); e != nil {
			return e
		}
	}

	// populate based on requested claims.
	if opt.RequestedClaims == nil {
		return nil
	}

	for _, specs := range opt.ClaimsFormula {
		filter := func(name string, spec ClaimSpec) bool {
			requested, ok := opt.RequestedClaims.Get(name)
			return !ok || !requested.Essential()
		}
		if e := populateWithSpecs(ctx, claims, specs, &opt, filter); e != nil {
			return e
		}
	}

	return nil
}

type claimSpecFilter func(name string, spec ClaimSpec) (exclude bool)

func populateWithSpecs(ctx context.Context, claims oauth2.Claims, specs map[string]ClaimSpec, opt *FactoryOption, filter claimSpecFilter) error {
	for c, spec := range specs {
		if c == "" || filter != nil && filter(c, spec) {
			continue
		}

		v, e := spec.Calculate(ctx, opt)
		if e != nil && spec.Required(ctx, opt) {
			return fmt.Errorf("unable to create claim [%s]: %v", c, e)
		} else if e != nil {
			continue
		}

		// check type and assign
		if e := safeSet(claims, c, v); e != nil {
			return e
		}
	}
	return nil
}

func safeSet(claims oauth2.Claims, claim string, value interface{}) (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		if e, ok := r.(error); ok {
			err = fmt.Errorf("unable to create claim [%s]: %v", claim, e)
		} else {
			err = fmt.Errorf("unable to create claim [%s]: %v", claim, r)
		}
	}()

	claims.Set(claim, value)
	return nil
}

/*************************
	helpers
 *************************/
func tryReloadAccount(ctx context.Context, opt *FactoryOption) security.Account {
	if acct, ok := ctx.Value(oauth2.CtxKeyAuthenticatedAccount).(security.Account); ok {
		return acct
	}

	if opt.AccountStore == nil {
		return nil
	}

	details, ok := opt.Source.Details().(security.UserDetails)
	if !ok {
		return nil
	}

	user, e := opt.AccountStore.LoadAccountById(ctx, details.UserId())
	if e != nil {
		return nil
	}

	// cache it in context if possible
	if mc, ok := ctx.(utils.MutableContext); ok {
		mc.Set(oauth2.CtxKeyAuthenticatedAccount, user)
	}
	return user
}
